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
	"log/syslog"
	"math"
	"os"
	"os/exec"
	"path"
	"strconv"
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
	ConfigFilePath string `toml:"-"`
	Username       string `toml:"username"`
	Password       string `toml:"password"`
	Debug          bool   `toml:"debug"`
	Foreground     bool   `toml:"foreground"`
	AllowOther     bool   `toml:"allow-other"`
	UID            int64  `toml:"uid"`
	GID            int64  `toml:"gid"`
}

func (c *config) loadMountOptions(options string) {
	optionsMap := parseOptions(options)
	for key, val := range optionsMap {
		switch key {
		case "rw", "ro", "dev", "nodev", "suid", "nosuid":
			// ignoring these options
		case "config":
			c.ConfigFilePath = val
		case "username":
			c.Username = val
		case "password":
			c.Password = val
		case "debug":
			c.Debug = true
		case "foreground":
			c.Foreground = true
		case "allow-other", "allow_other":
			c.AllowOther = true
		case "uid":
			if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
				if intVal > math.MaxUint32 {
					log.Fatalf("UID too big (Max allowed uid is %v): %v\n", int64(math.MaxUint32), intVal)
				}

				c.UID = intVal
			} else {
				log.Fatalf("Failed to parse uid mount option: '%v'\n", err)
			}
		case "gid":
			if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
				if intVal > math.MaxUint32 {
					log.Fatalf("GID too big (Max allowed gid is %v): %v\n", int64(math.MaxUint32), intVal)
				}

				c.UID = intVal
			} else {
				log.Fatalf("Failed to parse gid mount option: '%v'\n", err)
			}
		default:
			log.Fatalf("Unknown mount option: '%v'\n", key)
		}
	}
}

func parseOptions(options string) map[string]string {
	optionsMap := make(map[string]string)

	var off int

	var inQuotes bool

	for i := 0; i < len(options); i++ {
		if options[i] == ',' && !inQuotes {
			parseOption(options[off:i], optionsMap)
			off = i + 1
		} else if options[i] == '"' {
			if !inQuotes {
				inQuotes = true
			} else if i > 0 && options[i-1] != '\\' {
				inQuotes = false
			}
		}
	}
	parseOption(options[off:], optionsMap)

	return optionsMap
}

func parseOption(s string, optionsMap map[string]string) {
	if len(s) == 0 {
		return
	}

	if strings.Contains(s, "=") {
		kv := strings.SplitN(s, "=", 2)
		optionsMap[kv[0]] = kv[1]
	} else {
		optionsMap[s] = ""
	}
}

func parseConfig() config {
	c := config{
		UID: -1,
		GID: -1,
	}

	var options string

	// Disable sorting of flags
	flag.CommandLine.SortFlags = false
	flag.StringVarP(&c.ConfigFilePath, "config", "c", c.ConfigFilePath, "config file path")
	flag.StringVarP(&c.Username, "username", "u", c.Username, "mycloud.com username")
	flag.StringVarP(&c.Password, "password", "p", c.Password, "mycloud.com password")
	flag.BoolVarP(&c.AllowOther, "allow-other", "a", c.AllowOther, "allow other users")
	flag.Int64VarP(&c.UID, "uid", "U", c.UID, "set the owner of the files in the filesystem")
	flag.Int64VarP(&c.GID, "gid", "G", c.GID, "set the group of the files in the filesystem")
	flag.BoolVarP(&c.Foreground, "foreground", "f", c.Foreground, "do not demonize")
	flag.BoolVarP(&c.Debug, "debug", "d", c.Debug, "activate debug output (implies --foreground)")
	flag.StringVarP(&options, "options", "o", "", "mount options")
	help := flag.BoolP("help", "h", false, "display this help and exit")

	// The `options` flag is only to support being called by mount.
	// We hide it in the user help
	flag.Lookup("options").Hidden = true
	flag.Lookup("uid").DefValue = "disabled"
	flag.Lookup("gid").DefValue = "disabled"

	flag.Parse()

	if *help {
		c.printUsage()
		os.Exit(0)
	}

	c.loadMountOptions(options)

	if c.ConfigFilePath == "" {
		if file, err := os.OpenFile(defaultConfigFilePath, os.O_RDONLY, 0666); err == nil {
			_ = file.Close()
			c.ConfigFilePath = defaultConfigFilePath
		}
	}

	c.loadConfigFile()
	flag.Parse()
	c.validateConfig()

	return c
}

func (c *config) loadConfigFile() {
	if c.ConfigFilePath == "" {
		return
	}

	if _, err := os.Stat(c.ConfigFilePath); err != nil {
		log.Fatalf("Configuration file %v doesn't exist or is unreadable.\n", c.ConfigFilePath)
	}

	if _, err := toml.DecodeFile(c.ConfigFilePath, c); err != nil {
		log.Fatalf("Error: %v\n", err)
	}
}

func (c *config) validateConfig() {
	if c.Username == "" {
		log.Fatalf("Username is required. Set it in configuration file or specify it with --username flag.\n")
	}

	if c.Password == "" {
		log.Fatalf("Password is required. Set it in configuration file or specify it with --password flag.\n")
	}

	if c.UID < 0 {
		c.UID = int64(syscall.Getuid())
	}

	if c.GID < 0 {
		c.GID = int64(syscall.Getgid())
	}

	// Debugging implies running in foreground
	c.Foreground = c.Foreground || c.Debug
}

func (c *config) printUsage() {
	_, _ = fmt.Fprintf(os.Stderr, "Usage: %v [flags] deviceName[:devicePath] mountpoint\n", path.Base(os.Args[0]))

	flag.PrintDefaults()
}

func main() {
	config := parseConfig()

	if len(flag.Args()) <= mountPointPos {
		config.printUsage()
		os.Exit(1)
	}

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

	if !config.Foreground {
		if os.Getppid() > 1 {
			if _, err := demonize(); err != nil {
				log.Fatalf("Error demonising: %s", err)
			}

			os.Exit(0)
		}

		// Logging output must go to syslog as stderr is not available in a daemon process
		redirectOutputToSyslog()
	}

	if err := mount(file, source, mountPoint, config); err != nil {
		log.Fatal(err)
	}
}

func redirectOutputToSyslog() {
	syslogWriter, e := syslog.New(syslog.LOG_NOTICE, "mchfuse")
	if e == nil {
		log.SetOutput(syslogWriter)
	}

	// Redirect standard file descriptors to `/dev/null`
	os.Stdin = os.NewFile(uintptr(syscall.Stdin), os.DevNull)
	os.Stdout = os.NewFile(uintptr(syscall.Stdout), os.DevNull)
	os.Stderr = os.NewFile(uintptr(syscall.Stderr), os.DevNull)
}

func demonize() (int, error) {
	executable, err := os.Executable()
	if err != nil {
		return 0, err
	}

	args := os.Args[1:]
	cmd := exec.Command(executable, args...)
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Setsid is used to detach the process from the parent (normally a shell)
		//
		// The disowning of a child process is accomplished by executing the system call
		// setpgrp() or setsid(), (both of which have the same functionality) as soon as
		// the child is forked. These calls create a new process session group, make the
		// child process the session leader, and set the process group ID to the process
		// ID of the child. https://bsdmag.org/unix-kernel-system-calls/
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	return cmd.Process.Pid, nil
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
