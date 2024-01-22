package s3

import (
	"fmt"
	"log"
	"os"

	"github.com/golang/glog"
)

const (
	goofysCmd = "goofys"
)

// Implements Mounter
type goofysMounter struct {
	bucket          *bucket
	endpoint        string
	region          string
	accessKeyID     string
	secretAccessKey string
	volumeID        string
	readonly        bool
}

func newGoofysMounter(b *bucket, cfg *Config, volume string) (Mounter, error) {
	glog.V(3).Infof("Mounting using goofys volume %s", volume)
	//TODO we need to handle regions as well
	//region := cfg.Region
	//// if endpoint is set we need a default region
	//if region == "" && cfg.Endpoint != "" {
	//	region = defaultRegion
	//}
	return &goofysMounter{
		bucket:          b,
		endpoint:        cfg.Endpoint,
		region:          cfg.Region,
		accessKeyID:     cfg.AccessKeyID,
		secretAccessKey: cfg.SecretAccessKey,
		readonly:        cfg.Readonly,
		volumeID:        volume,
	}, nil
}

func (goofys *goofysMounter) Stage(stageTarget string) error {
	return nil
}

func (goofys *goofysMounter) Unstage(stageTarget string) error {
	return nil
}

func (goofys *goofysMounter) Mount(source string, target string) error {
	glog.V(3).Infof("Mounting using goofys!")

	if err := writes3fsPassGoofy(goofys); err != nil {
		return err
	}
	args := []string{
		fmt.Sprintf("--endpoint=%s", goofys.endpoint),
		"--type-cache-ttl", "1s",
		"--stat-cache-ttl", "1s",
		"--dir-mode", "0777",
		"--file-mode", "0777",
		"--http-timeout", "5m",
		//fmt.Sprintf("--cheap=%s", os.Getenv("cheap")),
		"-o", "allow_other",
	}
	if goofys.accessKeyID != "" && goofys.secretAccessKey != "" {
		args = append(args, fmt.Sprintf("--profile=%s", goofys.volumeID))
	}
	if goofys.region != "" {
		args = append(args, "--region", goofys.region)
	}
	if goofys.readonly {
		args = append(args, "-o", "ro")
	}
	if goofys.bucket.Folder == "" {
		glog.V(4).Infof("This bucket %s contains no folder prefixes", goofys.bucket.Name)
		args = append(args,
			fmt.Sprintf("%s", goofys.bucket.Name),
			fmt.Sprintf("%s", target))
	} else {
		glog.V(4).Infof("This bucket %s contains folder prefix %s", goofys.bucket.Name, goofys.bucket.Folder)
		args = append(args,
			fmt.Sprintf("%s:%s", goofys.bucket.Name, goofys.bucket.Folder),
			fmt.Sprintf("%s", target))
	}
	return fuseMount(target, goofysCmd, args)

}

func writes3fsPassGoofy(goofys *goofysMounter) error {
	awsPath := fmt.Sprintf("%s/.aws", os.Getenv("HOME"))
	if _, err := os.Stat(awsPath); os.IsNotExist(err) {
		mkdir_err := os.Mkdir(awsPath, 0700)
		if mkdir_err != nil {
			return mkdir_err
		}
	}

	pwFileName := fmt.Sprintf("%s/.aws/credentials", os.Getenv("HOME"))
	f, err := os.OpenFile(pwFileName,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	textToAdd := "[" + goofys.volumeID + "]\naws_access_key_id = " + goofys.accessKeyID + "\naws_secret_access_key =" + goofys.secretAccessKey + "\n"
	if _, err := f.WriteString(textToAdd); err != nil {
		log.Println(err)
	}
	return nil
}
