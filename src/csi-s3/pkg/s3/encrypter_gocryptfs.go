package s3

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

type gocryptfsEncrypter struct{}

func (enc *gocryptfsEncrypter) MountEncrypt(source string, target string, pass string) error {
	targetDir := filepath.Dir(target)
	passFile := filepath.Join(targetDir, "pass")

	err := CreateTextFile(passFile, pass)
	if err != nil {
		return err
	}

	args := []string{
		"-passfile", passFile,
		source,
		target,
	}

	err = fuseMount(target, gocryptfsCmd, args)
	if err != nil {
		return err
	}

	err = DeleteFile(passFile)
	if err != nil {
		return err
	}

	configFile := filepath.Join(source, "gocryptfs.conf")
	if !FileExists(configFile) {
		err := enc.initialize(target, passFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func (enc *gocryptfsEncrypter) initialize(target string, passFile string) error {
	args := []string{
		"-init",
		"-passfile", passFile,
		target,
	}

	cmd := exec.Command(gocryptfsCmd, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error gocryptfs initialize command: %s\nargs: %s\noutput: %s", gocryptfsCmd, args, out)
	}

	return nil
}
