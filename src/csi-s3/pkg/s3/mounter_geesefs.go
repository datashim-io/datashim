package s3

import (
	"fmt"
	"log"
	"os"

	"github.com/golang/glog"
)

const (
	geesefsCmd = "geesefs"
)

// Implements Mounter
type geesefsMounter struct {
	bucket          *bucket
	endpoint        string
	region          string
	accessKeyID     string
	secretAccessKey string
	volumeID        string
	readonly        bool
}

func newGeesefsMounter(b *bucket, cfg *Config, volume string) (Mounter, error) {
	glog.V(3).Infof("Mounting using geesefs volume %s", volume)
	//TODO we need to handle regions as well
	//region := cfg.Region
	//// if endpoint is set we need a default region
	//if region == "" && cfg.Endpoint != "" {
	//	region = defaultRegion
	//}
	return &geesefsMounter{
		bucket:          b,
		endpoint:        cfg.Endpoint,
		region:          cfg.Region,
		accessKeyID:     cfg.AccessKeyID,
		secretAccessKey: cfg.SecretAccessKey,
		readonly:        cfg.Readonly,
		volumeID:        volume,
	}, nil
}

func (geesefs *geesefsMounter) Stage(stageTarget string) error {
	return nil
}

func (geesefs *geesefsMounter) Unstage(stageTarget string) error {
	return nil
}

func (geesefs *geesefsMounter) Mount(source string, target string) error {
	glog.V(3).Infof("Mounting using geesefs!")

	if err := writes3fsPassGeesefs(geesefs); err != nil {
		return err
	}
	args := []string{
		fmt.Sprintf("--endpoint=%s", geesefs.endpoint),
		"--stat-cache-ttl", "1s",
		"--dir-mode", "0777",
		"--file-mode", "0777",
		"--http-timeout", "5m",
		//fmt.Sprintf("--cheap=%s", os.Getenv("cheap")),
		"-o", "allow_other",
	}
	if geesefs.accessKeyID != "" && geesefs.secretAccessKey != "" {
		args = append(args, fmt.Sprintf("--profile=%s", geesefs.volumeID))
	}
	if geesefs.region != "" {
		args = append(args, "--region", geesefs.region)
	}
	if geesefs.readonly {
		args = append(args, "-o", "ro")
	}
	if geesefs.bucket.Folder == "" {
		glog.V(4).Infof("This bucket %s contains no folder prefixes", geesefs.bucket.Name)
		args = append(args,
			fmt.Sprintf("%s", geesefs.bucket.Name),
			fmt.Sprintf("%s", target))
	} else {
		glog.V(4).Infof("This bucket %s contains folder prefix %s", geesefs.bucket.Name, geesefs.bucket.Folder)
		args = append(args,
			fmt.Sprintf("%s:%s", geesefs.bucket.Name, geesefs.bucket.Folder),
			fmt.Sprintf("%s", target))
	}
	return fuseMount(target, geesefsCmd, args)

}

func writes3fsPassGeesefs(geesefs *geesefsMounter) error {
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
	textToAdd := "[" + geesefs.volumeID + "]\naws_access_key_id = " + geesefs.accessKeyID + "\naws_secret_access_key =" + geesefs.secretAccessKey + "\n"
	if _, err := f.WriteString(textToAdd); err != nil {
		log.Println(err)
	}
	return nil
}
