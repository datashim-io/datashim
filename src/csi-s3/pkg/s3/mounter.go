package s3

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/golang/glog"
	mount "k8s.io/mount-utils"
)

// Mounter interface which can be implemented
// by the different mounter types
type Mounter interface {
	Stage(stagePath string) error
	Unstage(stagePath string) error
	Mount(source string, target string) error
}

const (
	s3fsMounterType     = "s3fs"
	goofysMounterType   = "goofys"
	s3backerMounterType = "s3backer"
	rcloneMounterType   = "rclone"
	mounterTypeKey      = "mounter"
)

// newMounter returns a new mounter depending on the mounterType parameter
func newMounter(bucket *bucket, cfg *Config, volumeId string) (Mounter, error) {
	mounter := bucket.Mounter
	// Fall back to mounterType in cfg
	if len(bucket.Mounter) == 0 {
		mounter = cfg.Mounter
	}
	switch mounter {
	case s3fsMounterType:
		return newS3fsMounter(bucket, cfg)

	case goofysMounterType:
		return newGoofysMounter(bucket, cfg, volumeId)

	case s3backerMounterType:
		return newS3backerMounter(bucket, cfg)

	case rcloneMounterType:
		return newRcloneMounter(bucket, cfg)

	default:
		// default to s3fsMounter
		return newS3fsMounter(bucket, cfg)
	}
}

func fuseMount(path string, command string, args []string) error {
	cmd := exec.Command(command, args...)
	glog.V(3).Infof("Mounting fuse with command: %s and args: %s", command, args)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error fuseMount command: %s\nargs: %s\noutput: %s", command, args, out)
	}

	return waitForMount(path, 10*time.Second)
}

func fuseUnmount(path string) error {
	// Try to unmount, but don't fail early - we need to cleanup the fuse process regardless
	unmountErr := mount.New("").Unmount(path)
	if unmountErr != nil {
		glog.Warningf("Unmount failed for path %s: %s, will still attempt to cleanup fuse process", path, unmountErr)
	}

	// as fuse quits immediately, we will try to wait until the process is done
	// This cleanup is critical even if unmount failed, to prevent orphaned processes
	process, err := findFuseMountProcess(path)
	if err != nil {
		glog.Errorf("Error getting PID of fuse mount: %s", err)
		return unmountErr // Return the original unmount error if we had one
	}
	if process == nil {
		glog.Warningf("Unable to find PID of fuse mount %s, it must have finished already", path)
		return unmountErr // Return the original unmount error if we had one
	}
	glog.Infof("Found fuse pid %v of mount %s, checking if it still runs", process.Pid, path)
	processErr := waitForProcess(process, 1)

	// Return combined error information
	if unmountErr != nil && processErr != nil {
		return fmt.Errorf("unmount failed: %v, process cleanup failed: %v", unmountErr, processErr)
	} else if unmountErr != nil {
		return unmountErr
	}
	return processErr
}
