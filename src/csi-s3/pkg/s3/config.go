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
	Readonly        bool
	Provision       bool
	Encrypter       string
	EncryptionKey   string
}
