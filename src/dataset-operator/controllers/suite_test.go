/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	datasetsv1alpha1 "github.com/datashim-io/datashim/src/dataset-operator/api/v1alpha1"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	mock_driver "github.com/kubernetes-csi/csi-test/v5/driver"
	mock_utils "github.com/kubernetes-csi/csi-test/v5/utils"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestMockCSIDriver(t *testing.T) {
	m := gomock.NewController(&mock_utils.SafeGoroutineTester{})
	defer m.finish()

 	driver := mock_driver.NewMockControllerServer(m)

	defaultVolumeID := "vol1"
	defaultNodeID := "node1"

	defaultCaps := &csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{},
		},
		AccessMode: &csi.VolumeCapability_AccessMode{
			Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
		},
	}
	
	publishVolumeInfo := map[string]string{
		"first": "foo",
		"second": "bar",
		"third": "baz",
	}

	defaultRequest := &csi.ControllerPublishVolumeRequest{
		PublishContext: publishVolumeInfo,
	}

	driver.EXPECT().ControllerPublishVolume(gomock.Any(), pbMatch(default))
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = datasetsv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

})

var _ = Describe("Create Dataset", func() {
	BeforeEach(func() {
		dataset = &datasetsv1alpha1.Dataset{
			TypeMeta: v1.TypeMeta{
				Kind:       "Dataset",
				APIVersion: "com.ie.ibm.hpsys/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-dataset",
				Namespace: "default",
			},
			Spec:   datasetsv1alpha1.DatasetSpec{
				Local:   map[string]string{
					"type":    "HOST",
					"path":    "/tmp/tmp123",
					"hostPathType": "CreateNew"
				},
			},
		}
	})

	
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
