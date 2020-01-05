package offline

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/jstaf/onedriver/graph"
	"github.com/jstaf/onedriver/logger"
	log "github.com/sirupsen/logrus"
)

const (
	mountLoc = "mount"
	TestDir  = mountLoc + "/onedriver_tests"
)

var auth *graph.Auth

// Like the graph package, but designed for running tests offline.
func TestMain(m *testing.M) {
	os.Chdir("..")
	// attempt to unmount regardless of what happens (in case previous tests
	// failed and didn't clean themselves up)
	exec.Command("fusermount", "-uz", mountLoc).Run()
	os.Mkdir(mountLoc, 0755)

	auth := graph.Authenticate()
	inode, err := graph.GetItem("root", auth)
	if inode != nil || !graph.IsOffline(err) {
		fmt.Println("These tests must be run offline.")
		os.Exit(1)
	}

	logFile, _ := os.OpenFile("offline_tests.log", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0644)
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetReportCaller(true)
	log.SetFormatter(logger.LogrusFormatter())
	log.SetLevel(log.DebugLevel)

	root := graph.NewFS("test.db", 5*time.Second)
	second := time.Second
	server, _ := fs.Mount(mountLoc, root, &fs.Options{
		EntryTimeout: &second,
		AttrTimeout:  &second,
		MountOptions: fuse.MountOptions{
			Name:          "onedriver",
			FsName:        "onedriver",
			DisableXAttrs: true,
			MaxBackground: 1024,
		},
	})

	// setup sigint handler for graceful unmount on interrupt/terminate
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go graph.UnmountHandler(sigChan, server)

	// mount fs in background thread
	go server.Serve()

	code := m.Run()

	server.Unmount()
	os.Exit(code)
}
