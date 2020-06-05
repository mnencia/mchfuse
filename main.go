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

package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	flag "github.com/spf13/pflag"

	"github.com/mnencia/mchfuse/fsnode"
	"github.com/mnencia/mchfuse/mch"
)

const (
	defaultConfigFilePath = "/etc/mchfuse.conf"
)

const (
	sourcePos = iota
	mountPointPos
)

type config struct {
	Username   string `toml:"username"`
	Password   string `toml:"password"`
	Debug      bool   `toml:"debug"`
	AllowOther bool   `toml:"allow-other"`
	UID        int    `toml:"uid"`
	GID        int    `toml:"gid"`
}

func parseConfig() config {
	configFilePath := flag.StringP("config", "c", "", "config file path")
	username := flag.StringP("username", "u", "", "mycloud.com username ")
	password := flag.StringP("password", "p", "", "mycloud.com password")
	debug := flag.BoolP("debug", "d", false, "activate debug output")
	allowOther := flag.BoolP("allow-other", "a", false, "allow other users")
	uid := flag.IntP("uid", "U", -1, "set the owner of the files in the filesystem")
	gid := flag.IntP("gid", "G", -1, "set the group of the files in the filesystem")
	flag.Parse()

	c := loadConfigFile(*configFilePath)

	if *username != "" {
		c.Username = *username
	}

	if *password != "" {
		c.Password = *password
	}

	if *debug {
		c.Debug = true
	}

	if *allowOther {
		c.AllowOther = true
	}

	if *uid > 0 {
		c.UID = *uid
	}

	if *gid > 0 {
		c.GID = *gid
	}

	return c
}

func loadConfigFile(configFilePath string) config {
	c := config{
		UID: -1,
		GID: -1,
	}

	if configFilePath != "" {
		if _, err := os.Stat(configFilePath); err != nil {
			fmt.Printf("Configuration file %v doesn't exist or is unreadable.\n", configFilePath)
			os.Exit(1)
		}
	} else {
		if _, err := os.Stat(defaultConfigFilePath); err == nil {
			configFilePath = defaultConfigFilePath
		}
	}

	if configFilePath != "" {
		if _, err := toml.DecodeFile(configFilePath, &c); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}

	return c
}

func validateConfig(c config) config {
	if c.Username == "" {
		fmt.Printf("Username is required. Set it in configuration file or specify it with --username flag.\n")
		os.Exit(1)
	}

	if c.Password == "" {
		fmt.Printf("Password is required. Set it in configuration file or specify it with --password flag.\n")
		os.Exit(1)
	}

	if c.UID < 0 {
		c.UID = syscall.Getuid()
	}

	if c.GID < 0 {
		c.GID = syscall.Getgid()
	}

	return c
}

func main() {
	config := parseConfig()

	if len(flag.Args()) <= mountPointPos {
		fmt.Printf("Usage: %v [flags] deviceName[:devicePath] mountpoint\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
		os.Exit(1)
	}

	config = validateConfig(config)

	source := flag.Arg(sourcePos)
	mountPoint := flag.Arg(mountPointPos)

	var deviceName, devicePath string

	i := strings.Index(source, ":")
	if i > -1 {
		deviceName, devicePath = source[:i], source[i+1:]
	} else {
		deviceName = source
	}

	client, err := mch.Login(config.Username, config.Password)
	if err != nil {
		log.Fatalf("Failure signing in My Cloud Home account: %s", err)
	}

	deviceList, err := client.DeviceInfo()
	if err != nil {
		log.Fatalf("Failure retrieving device list: %s", err)
	}

	device := deviceList.Find(deviceName)
	if device == nil {
		available := make([]string, 0)
		for _, device := range deviceList.Data {
			available = append(available, device.Name)
		}

		log.Fatalf("Unknown device \"%s\" (available devices: %s)", deviceName, strings.Join(available, ", "))
	}

	file, err := device.GetFileByPath(devicePath)
	if err != nil {
		log.Fatalf("Failure searching for path %s: %s", devicePath, err)
	}

	if err := mount(file, source, mountPoint, config); err != nil {
		log.Fatal(err)
	}
}

func mount(file *mch.File, source, mountPoint string, config config) error {
	mchRoot := fsnode.NewMCHNode(file)
	sec := time.Second
	mountOpts := &fs.Options{
		MountOptions: fuse.MountOptions{
			AllowOther: config.AllowOther,
			Debug:      config.Debug,
			FsName:     source,
			Name:       "mchfuse",
		},
		UID:          uint32(config.UID),
		GID:          uint32(config.GID),
		AttrTimeout:  &sec,
		EntryTimeout: &sec,
	}
	// Mount the file system
	server, err := fs.Mount(mountPoint, mchRoot, mountOpts)
	if err != nil {
		return err
	}
	// Serve the file system, until unmounted by calling fusermount -u
	server.Wait()

	return nil
}
