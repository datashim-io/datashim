package admissioncontroller

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAdmissioncontroller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Admissioncontroller Suite")
}
