package s3

type Encrypter interface {
	MountEncrypt(source string, target string, pass string) error
}

const (
	gocryptfsCmd = "gocryptfs"
)

func NewEncrypter(encrypter string) (Encrypter, error) {
	switch encrypter {
	case gocryptfsCmd:
		return &gocryptfsEncrypter{}, nil
	default:
		return &gocryptfsEncrypter{}, nil
	}
}
