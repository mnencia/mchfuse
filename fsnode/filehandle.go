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
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type MCHFileHandle struct {
	fs.FileHandle
	node *MCHNode
}

var (
	_ = (fs.FileWriter)((*MCHFileHandle)(nil))
	_ = (fs.FileReader)((*MCHFileHandle)(nil))
)

func (mf *MCHFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	size := int64(mf.node.file.Size)
	if off > size {
		// ENXIO = no such address.
		return nil, syscall.Errno(int(syscall.ENXIO))
	}

	end := off + int64(len(dest))
	if end > size {
		dest = dest[:size-off]
	}

	read, err := mf.node.file.Read(dest, off)
	if err != nil {
		return nil, syscall.EIO
	}

	return fuse.ReadResultData(dest[:read]), fs.OK
}

func (mf *MCHFileHandle) Write(ctx context.Context, data []byte, off int64) (written uint32, errno syscall.Errno) {
	if err := mf.node.file.Write(data, off); err != nil {
		return 0, syscall.EIO
	}

	return uint32(len(data)), fs.OK
}
