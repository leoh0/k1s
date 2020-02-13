package main

import (
	"flag"
	"os"
	"runtime"

	"github.com/leoh0/binctr/container"
	"github.com/opencontainers/runc/libcontainer"
	_ "github.com/opencontainers/runc/libcontainer/nsenter"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
)

const (
	defaultRoot      = "/tmp/k1s"
	defaultRootfsDir = "rootfs"
)

var (
	containerID string
	root        string

	file      string
	dir       string
	shortpath string
)

func init() {
	// Parse flags
	flag.StringVar(&containerID, "id", "k1s", "container ID")
	flag.StringVar(&root, "root", defaultRoot, "root directory of container state, should be tmpfs")

	flag.Usage = func() {
		flag.PrintDefaults()
	}

	flag.Parse()
}

//go:generate go run generate.go
func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runInit()
		return
	}

	// Create a new container spec with the following options.
	opts := container.SpecOpts{
		Rootless: false,
		Terminal: false,
		Args: []string{
			"/kube/run.sh",
		},
		Mounts: []specs.Mount{
			{
				Destination: "/lib/modules",
				Type:        "bind",
				Source:      "/lib/modules",
				Options:     []string{"ro", "rbind", "rprivate"},
			},
			{
				Destination: "/sys/fs/cgroup",
				Type:        "bind",
				Source:      "/sys/fs/cgroup",
				Options:     []string{"ro", "rbind", "rprivate"},
			},
		},
	}
	spec := container.Spec(opts)

	// Initialize the container object.
	c := &container.Container{
		ID:               containerID,
		Spec:             spec,
		Root:             root,
		Rootless:         false,
		Detach:           false,
		NoPivotRoot:      true,
		UseSystemdCgroup: false,
		HostNetwork:      true,
		ShareIPC:         true,
	}

	// Unpack the rootfs.
	if err := c.UnpackRootfs(defaultRootfsDir, Asset); err != nil {
		logrus.Fatal(err)
	}

	// Run the container.
	status, err := c.Run()
	if err != nil {
		logrus.Fatal(err)
	}

	// Remove the rootfs after the container has exited.
	if err := os.RemoveAll(defaultRootfsDir); err != nil {
		logrus.Warnf("removing rootfs failed: %v", err)
	}

	// Exit with the container's exit status.
	os.Exit(status)
}

func runInit() {
	runtime.GOMAXPROCS(1)
	runtime.LockOSThread()
	factory, _ := libcontainer.New("")
	if err := factory.StartInitialization(); err != nil {
		// as the error is sent back to the parent there is no need to log
		// or write it to stderr because the parent process will handle this
		os.Exit(1)
	}
	panic("libcontainer: container init failed to exec")
}
