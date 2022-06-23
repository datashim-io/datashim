package s3

import (
	"fmt"
	"os"
)

// Implements Mounter
type s3fsMounter struct {
	bucket        *bucket
	readonly      bool
	url           string
	region        string
	pwFileContent string
}

const (
	s3fsCmd = "s3fs"
)

func newS3fsMounter(b *bucket, cfg *Config) (Mounter, error) {
	return &s3fsMounter{
		bucket:        b,
		url:           cfg.Endpoint,
		region:        cfg.Region,
		pwFileContent: cfg.AccessKeyID + ":" + cfg.SecretAccessKey,
		readonly:      cfg.Readonly,
	}, nil
}

func (s3fs *s3fsMounter) Stage(stageTarget string) error {
	return nil
}

func (s3fs *s3fsMounter) Unstage(stageTarget string) error {
	return nil
}

func (s3fs *s3fsMounter) Mount(source string, target string) error {
	if err := writes3fsPass(s3fs.pwFileContent); err != nil {
		return err
	}
	args := []string{
		fmt.Sprintf("%s", s3fs.bucket.Name),
		fmt.Sprintf("%s", target),
		"-o", "use_path_request_style",
		"-o", fmt.Sprintf("url=%s", s3fs.url),
		"-o", fmt.Sprintf("endpoint=%s", s3fs.region),
		"-o", "allow_other",
		"-o", "umask=0000",
	}
	if s3fs.readonly {
		args = append(args, "-o", "ro")
	}
	return fuseMount(target, s3fsCmd, args)
}

func writes3fsPass(pwFileContent string) error {
	pwFileName := fmt.Sprintf("%s/.passwd-s3fs", os.Getenv("HOME"))
	pwFile, err := os.OpenFile(pwFileName, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	_, err = pwFile.WriteString(pwFileContent)
	if err != nil {
		return err
	}
	pwFile.Close()
	return nil
}
