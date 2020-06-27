/*
Copyright 2020 Marco Nenciarini <mnencia@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fsnode

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	"github.com/mnencia/mchfuse/mch"
)

type MCHNode struct {
	fs.Inode
	file *mch.File
}

var _ = (fs.InodeEmbedder)((*MCHNode)(nil))
var _ = (fs.NodeReaddirer)((*MCHNode)(nil))
var _ = (fs.NodeLookuper)((*MCHNode)(nil))
var _ = (fs.NodeGetattrer)((*MCHNode)(nil))
var _ = (fs.NodeGetxattrer)((*MCHNode)(nil))
var _ = (fs.NodeSetxattrer)((*MCHNode)(nil))
var _ = (fs.NodeOpener)((*MCHNode)(nil))
var _ = (fs.NodeUnlinker)((*MCHNode)(nil))
var _ = (fs.NodeRmdirer)((*MCHNode)(nil))
var _ = (fs.NodeRenamer)((*MCHNode)(nil))
var _ = (fs.NodeMkdirer)((*MCHNode)(nil))
var _ = (fs.NodeCreater)((*MCHNode)(nil))
var _ = (fs.NodeSetattrer)((*MCHNode)(nil))
var _ = (fs.NodeMknoder)((*MCHNode)(nil))

var ErrorInvalidFilesystemStatus = errors.New("invalid filesytem status")

func NewMCHNode(file *mch.File) *MCHNode {
	return &MCHNode{file: file}
}

func (mn *MCHNode) mode() uint32 {
	if mn.file.IsDirectory() {
		return fuse.S_IFDIR
	}

	return fuse.S_IFREG
}

func (mn *MCHNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	if err := mn.fetch(ctx); err != nil {
		return nil, syscall.EIO
	}

	children := mn.Children()

	r := make([]fuse.DirEntry, 0, len(children))
	for k, ch := range children {
		r = append(r, fuse.DirEntry{
			Mode: ch.Mode(),
			Name: k,
		})
	}

	return fs.NewListDirStream(r), 0
}

func (mn *MCHNode) fetch(ctx context.Context) error {
	fileList, err := mn.file.ListDirectory()
	if err != nil {
		return err
	}

	for name := range mn.Children() {
		if _, found := fileList[name]; !found {
			mn.RmChild(name)
		}
	}

	for name, info := range fileList {
		// Pin the variable
		info := info
		if err := mn.updateChild(ctx, name, &info); err != nil {
			return err
		}
	}

	return nil
}

func (mn *MCHNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	child := mn.GetChild(name)
	if child == nil {
		if err := mn.lookup(ctx, name); err != nil {
			return nil, syscall.EIO
		}

		child = mn.GetChild(name)
		if child == nil {
			return nil, syscall.ENOENT
		}
	}

	if childNode, ok := child.Operations().(*MCHNode); ok {
		childNode.getattr(&out.Attr)
	}

	return child, fs.OK
}

func (mn *MCHNode) lookup(ctx context.Context, name string) error {
	info, err := mn.file.LookupDirectory(name)
	if err != nil {
		return err
	}

	child := mn.GetChild(name)

	// If the target doesn't exist, make sure it is not present in the cache and return
	if info == nil {
		if child != nil {
			mn.RmChild(name)
		}

		return nil
	}

	if err := mn.updateChild(ctx, name, info); err != nil {
		return err
	}

	return nil
}

func (mn *MCHNode) updateChild(ctx context.Context, name string, info *mch.File) error {
	child := mn.GetChild(name)

	// If the target exists and we do not have it in cache, add it
	if child == nil {
		childNode := NewMCHNode(info)
		mn.AddChild(name,
			mn.NewInode(ctx, childNode, fs.StableAttr{
				Mode: childNode.mode(),
			}),
			false)

		return nil
	}

	// Compare the content of the cache
	if childNode, ok := child.Operations().(*MCHNode); ok {
		if !reflect.DeepEqual(childNode.file, info) {
			childNode.file = info
		}
	} else {
		return fmt.Errorf("got a child of type %T instead of expected *MCHNode: %w",
			child.Operations(), ErrorInvalidFilesystemStatus)
	}

	return nil
}

func (mn *MCHNode) Getattr(ctx context.Context, file fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	if err := mn.file.Refresh(); err != nil {
		return syscall.EIO
	}

	mn.getattr(&out.Attr)

	return fs.OK
}

func (mn *MCHNode) getattr(out *fuse.Attr) {
	if mn.file.IsDirectory() {
		out.Mode = 0755
	} else {
		out.Mode = 0644
	}

	out.Size = mn.file.Size
	mtime := time.Time(mn.file.MTime)
	ctime := time.Time(mn.file.CTime)
	out.SetTimes(&mtime, &mtime, &ctime)
}

func (mn *MCHNode) Getxattr(ctx context.Context, attr string, dest []byte) (uint32, syscall.Errno) {
	return 0, syscall.ENOSYS
}

func (mn *MCHNode) Setxattr(ctx context.Context, attr string, dest []byte, flags uint32) syscall.Errno {
	return syscall.ENOSYS
}

func (mn *MCHNode) Open(ctx context.Context, flags uint32) (file fs.FileHandle, fuseFlags uint32, code syscall.Errno) {
	return MCHFileHandle{node: mn}, 0, fs.OK
}

func (mn *MCHNode) Unlink(ctx context.Context, name string) syscall.Errno {
	if err := mn.lookup(ctx, name); err != nil {
		return syscall.EIO
	}

	child := mn.GetChild(name)
	if child == nil {
		return syscall.ENOENT
	}

	childNode := child.Operations().(*MCHNode)
	if err := childNode.file.Delete(); err != nil {
		return syscall.EIO
	}

	mn.RmChild(name)

	return 0
}

func (mn *MCHNode) Rmdir(ctx context.Context, name string) syscall.Errno {
	if err := mn.lookup(ctx, name); err != nil {
		return syscall.EIO
	}

	child := mn.GetChild(name)
	if child == nil {
		return syscall.ENOENT
	}

	childNode := child.Operations().(*MCHNode)

	if childNode.file.ChildCount > 0 {
		return syscall.ENOTEMPTY
	}

	if err := childNode.file.Delete(); err != nil {
		return syscall.EIO
	}

	mn.RmChild(name)

	return 0
}

func (mn *MCHNode) Rename(
	ctx context.Context,
	name string,
	newParent fs.InodeEmbedder,
	newName string,
	flags uint32,
) syscall.Errno {
	if flags&fs.RENAME_EXCHANGE > 0 {
		return syscall.EINVAL
	}

	if err := mn.lookup(ctx, name); err != nil {
		return syscall.EIO
	}

	src := mn.GetChild(name)
	if src == nil {
		return syscall.ENOENT
	}

	srcNode, ok := src.Operations().(*MCHNode)
	if !ok {
		return syscall.ENOSYS
	}

	newParentNode, ok := newParent.(*MCHNode)
	if !ok {
		return syscall.ENOSYS
	}

	if err := newParentNode.lookup(ctx, newName); err != nil {
		return syscall.EIO
	}

	dest := newParentNode.GetChild(newName)
	if dest != nil {
		return syscall.EEXIST
	}

	if err := srcNode.file.Rename(newParentNode.file, newName); err != nil {
		return syscall.EIO
	}

	return 0
}

func (mn *MCHNode) Mkdir(
	ctx context.Context,
	name string,
	mode uint32,
	out *fuse.EntryOut,
) (
	newInode *fs.Inode,
	errno syscall.Errno,
) {
	if err := mn.lookup(ctx, name); err != nil {
		errno = syscall.EIO
		return
	}

	if mn.GetChild(name) != nil {
		errno = syscall.EEXIST
		return
	}

	newFile, err := mn.file.CreateDirectory(name)
	if err != nil {
		errno = syscall.EIO
		return
	}

	newNode := NewMCHNode(newFile)
	newInode = mn.NewInode(ctx, newNode, fs.StableAttr{
		Mode: newNode.mode(),
	})
	mn.AddChild(name, newInode, true)

	var attr fuse.AttrOut

	errno = newNode.Getattr(ctx, nil, &attr)
	if errno != fs.OK {
		return
	}

	out.Attr = attr.Attr

	return newInode, fs.OK
}

func (mn *MCHNode) Create(
	ctx context.Context,
	name string,
	flags uint32,
	mode uint32,
	out *fuse.EntryOut,
) (
	node *fs.Inode,
	fh fs.FileHandle,
	fuseFlags uint32,
	errno syscall.Errno,
) {
	if err := mn.lookup(ctx, name); err != nil {
		return nil, nil, 0, syscall.EIO
	}

	if mn.GetChild(name) != nil {
		return nil, nil, 0, syscall.EEXIST
	}

	newFile, err := mn.file.Create(name)
	if err != nil {
		return nil, nil, 0, syscall.EIO
	}

	newNode := NewMCHNode(newFile)
	newInode := mn.NewInode(ctx, newNode, fs.StableAttr{
		Mode: newNode.mode(),
	})
	mn.AddChild(name, newInode, true)

	var attr fuse.AttrOut

	errno = newNode.Getattr(ctx, nil, &attr)
	if errno != fs.OK {
		return nil, nil, 0, errno
	}

	out.Attr = attr.Attr

	return newInode, MCHFileHandle{node: newNode}, 0, fs.OK
}

func (mn *MCHNode) Setattr(ctx context.Context, f fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	if size, ok := in.GetSize(); ok {
		if err := mn.file.Truncate(int64(size)); err != nil {
			return syscall.EIO
		}
	}

	changes := make(map[string]interface{})

	if mTime, ok := in.GetMTime(); ok {
		changes["mTime"] = mch.ISOTime(mTime)
	}

	if cTime, ok := in.GetCTime(); ok {
		changes["cTime"] = mch.ISOTime(cTime)
	}

	if len(changes) > 0 {
		if err := mn.file.SetMeta(changes); err != nil {
			return syscall.EIO
		}
	}

	return mn.Getattr(ctx, f, out)
}

func (*MCHNode) Mknod(
	ctx context.Context,
	name string,
	mode uint32,
	dev uint32,
	out *fuse.EntryOut,
) (
	*fs.Inode,
	syscall.Errno,
) {
	return nil, syscall.ENOSYS
}
