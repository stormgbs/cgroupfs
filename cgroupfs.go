package cgroupfs

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"

	"github.com/chenchun/cgroupfs/fs"
)

func Serve(mountPoint, cgroupDir string) error {
	c, err := fuse.Mount(
		mountPoint,
		fuse.FSName("cgroupfs"),
		fuse.Subtype("cgroupfs"),
		fuse.LocalVolume(),
		fuse.VolumeName("cgroup volume"),
		fuse.AllowOther(),
	)
	if err != nil {
		return err
	}
	defer c.Close()
	go handleStopSignals(mountPoint)

	var srv *fusefs.Server
	if os.Getenv("FUSE_DEBUG") != "" {
		srv = fusefs.New(c, &fusefs.Config{
			Debug: func(msg interface{}) {
				fmt.Printf("%s\n", msg)
			},
		})
	} else {
		srv = fusefs.New(c, nil)
	}

	err = srv.Serve(fs.FS{cgroupDir})
	if err != nil {
		return err
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		return err
	}

	return nil
}

func handleStopSignals(mountPoint string) {
	s := make(chan os.Signal, 10)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM, syscall.SIGSTOP)

	for range s {
		if err := fuse.Unmount(mountPoint); err != nil {
			log.Fatal("Error umounting %s: %s", mountPoint, err)
		}
		os.Exit(0)
	}
}
