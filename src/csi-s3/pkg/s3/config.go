package s3

// Config holds values to configure the driver
type Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Endpoint        string
	Mounter         string
	Bucket          string
	Folder          string
	RemoveOnDelete  bool
	Readonly        bool
	Provision       bool
}
