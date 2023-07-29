package admissioncontroller

import (
	testing "github.com/datashim-io/datashim/src/dataset-operator/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Test the mutation based on labels
var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
})

type testPodLabels struct {
	makeInputPodSpec          func() *corev1.Pod
	makeOutputPatchOperations func() []jsonpatch.JsonPatchOperation
}

var _ = DescribeTable("Pod is mutated correctly",
	func(tc *testPodLabels) {

		Expect(patchPodWithDatasetLabels(tc.makeInputPodSpec())).Should(Equal(tc.makeOutputPatchOperations()))

	},
	Entry("Pod with no volumes, 1 dataset label, useas mount -> 1 volume mount", &testPodLabels{
		makeInputPodSpec: func() *corev1.Pod {
			inputPod := testing.MakePod("test-1", "test").
				AddLabelToPodMetadata("dataset.0.id", "testds").
				AddLabelToPodMetadata("dataset.0.useas", "mount").
				AddContainerToPod(testing.MakeContainer("foo").Obj()).
				Obj()
			return &inputPod
		},
		makeOutputPatchOperations: func() []jsonpatch.JsonPatchOperation {
			patchArray := []jsonpatch.JsonPatchOperation{}
			patchArray = append(patchArray,
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetVolumeasPath(0).
					SetPVCasValue("testds").
					Obj(),
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetVolumeMountasPath("containers", 0, 0).
					SetVolumeMountasValue("testds").
					Obj())
			return patchArray
		},
	}),
	Entry("Pod with no volumes, 1 dataset label, useas configmap -> 1 config mount", &testPodLabels{
		makeInputPodSpec: func() *corev1.Pod {
			inputPod := testing.MakePod("test-1", "test").
				AddLabelToPodMetadata("dataset.0.id", "testds0").
				AddLabelToPodMetadata("dataset.0.useas", "configmap").
				AddContainerToPod(testing.MakeContainer("foo").Obj()).
				Obj()
			return &inputPod
		},
		makeOutputPatchOperations: func() []jsonpatch.JsonPatchOperation {
			patchArray := []jsonpatch.JsonPatchOperation{}
			patchArray = append(patchArray,
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetNewConfigMapRefasPath("containers", 0).
					AddConfigMapRefsToValue([]string{"testds0"}).
					AddSecretRefsToValue([]string{"testds0"}).
					Obj())
			return patchArray
		},
	}),
	Entry("Pod with no volumes, 2 dataset label, useas mount -> 2 volume mounts", &testPodLabels{
		makeInputPodSpec: func() *corev1.Pod {
			inputPod := testing.MakePod("test-1", "test").
				AddLabelToPodMetadata("dataset.0.id", "testds0").
				AddLabelToPodMetadata("dataset.0.useas", "mount").
				AddLabelToPodMetadata("dataset.1.id", "testds1").
				AddLabelToPodMetadata("dataset.1.useas", "mount").
				AddContainerToPod(testing.MakeContainer("foo").Obj()).
				Obj()
			return &inputPod
		},
		makeOutputPatchOperations: func() []jsonpatch.JsonPatchOperation {
			patchArray := []jsonpatch.JsonPatchOperation{}
			patchArray = append(patchArray,
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetVolumeasPath(0).
					SetPVCasValue("testds0").
					Obj(),
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetVolumeasPath(1).
					SetPVCasValue("testds1").
					Obj(),
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetVolumeMountasPath("containers", 0, 0).
					SetVolumeMountasValue("testds0").
					Obj(),
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetVolumeMountasPath("containers", 0, 1).
					SetVolumeMountasValue("testds1").
					Obj())
			return patchArray
		},
	}),
	Entry("Pod with no volumes, 2 dataset label, 1 useas mount, 1 useas configmap -> 1 volume, 1 configmap", &testPodLabels{
		makeInputPodSpec: func() *corev1.Pod {
			inputPod := testing.MakePod("test-1", "test").
				AddLabelToPodMetadata("dataset.0.id", "testds0").
				AddLabelToPodMetadata("dataset.0.useas", "mount").
				AddLabelToPodMetadata("dataset.1.id", "testds1").
				AddLabelToPodMetadata("dataset.1.useas", "configmap").
				AddContainerToPod(testing.MakeContainer("foo").Obj()).
				Obj()
			return &inputPod
		},
		makeOutputPatchOperations: func() []jsonpatch.JsonPatchOperation {
			patchArray := []jsonpatch.JsonPatchOperation{}
			patchArray = append(patchArray,
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetVolumeasPath(0).
					SetPVCasValue("testds0").
					Obj(),
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetVolumeMountasPath("containers", 0, 0).
					SetVolumeMountasValue("testds0").
					Obj(),
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetNewConfigMapRefasPath("containers", 0).
					AddConfigMapRefsToValue([]string{"testds1"}).
					AddSecretRefsToValue([]string{"testds1"}).
					Obj())
			return patchArray
		},
	}),
	Entry("Pod with 1 volumes, 1 dataset label (diff to existing), useas mount -> 2 volume mounts (1 existing, 1 new)", &testPodLabels{
		makeInputPodSpec: func() *corev1.Pod {
			inputPod := testing.MakePod("test-1", "test").
				AddLabelToPodMetadata("dataset.0.id", "testds0").
				AddLabelToPodMetadata("dataset.0.useas", "mount").
				AddVolumeToPod("testds").
				AddContainerToPod(testing.MakeContainer("foo").
					AddVolumeMount("/mnt/datasets/testds", "testv").Obj()).
				Obj()
			return &inputPod
		},
		makeOutputPatchOperations: func() []jsonpatch.JsonPatchOperation {
			patchArray := []jsonpatch.JsonPatchOperation{
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetVolumeasPath(1).
					SetPVCasValue("testds0").
					Obj(),
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetVolumeMountasPath("containers", 0, 1).
					SetVolumeMountasValue("testds0").
					Obj(),
			}
			return patchArray
		},
	}),
	Entry("Pod with 1 volumes, 1 dataset label (same as existing), useas mount -> 0 patches", &testPodLabels{
		makeInputPodSpec: func() *corev1.Pod {
			inputPod := testing.MakePod("test-1", "test").
				AddLabelToPodMetadata("dataset.0.id", "testds").
				AddLabelToPodMetadata("dataset.0.useas", "mount").
				AddVolumeToPod("testds").
				AddContainerToPod(testing.MakeContainer("foo").
					AddVolumeMount("/mnt/datasets/testds", "testds").Obj()).
				Obj()
			return &inputPod
		},
		makeOutputPatchOperations: func() []jsonpatch.JsonPatchOperation {
			patchArray := []jsonpatch.JsonPatchOperation{}
			return patchArray
		},
	}),
	Entry("Pod with 1 volumes, 1 dataset label, useas configmap -> 1 configmap", &testPodLabels{
		makeInputPodSpec: func() *corev1.Pod {
			inputPod := testing.MakePod("test-1", "test").
				AddLabelToPodMetadata("dataset.0.id", "testds0").
				AddLabelToPodMetadata("dataset.0.useas", "configmap").
				AddVolumeToPod("testds").
				AddContainerToPod(testing.MakeContainer("foo").
					AddVolumeMount("/mnt/datasets/testds", "testv").Obj()).
				Obj()
			return &inputPod
		},
		makeOutputPatchOperations: func() []jsonpatch.JsonPatchOperation {
			patchArray := []jsonpatch.JsonPatchOperation{
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetNewConfigMapRefasPath("containers", 0).
					AddConfigMapRefsToValue([]string{"testds0"}).
					AddSecretRefsToValue([]string{"testds0"}).
					Obj(),
			}
			return patchArray
		},
	}),
	Entry("Pod with no dataset volumes, 1 dataset label, useas mount, configmap -> 1 mount, 1 configmap for same dataset", &testPodLabels{
		makeInputPodSpec: func() *corev1.Pod {
			inputPod := testing.MakePod("test-1", "test").
				AddLabelToPodMetadata("dataset.0.id", "testds").
				AddLabelToPodMetadata("dataset.0.useas", "mount.configmap").
				AddContainerToPod(testing.MakeContainer("foo").
					Obj()).
				Obj()
			return &inputPod
		},
		makeOutputPatchOperations: func() []jsonpatch.JsonPatchOperation {
			patchArray := []jsonpatch.JsonPatchOperation{
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetVolumeasPath(0).
					SetPVCasValue("testds").
					Obj(),
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetVolumeMountasPath("containers", 0, 0).
					SetVolumeMountasValue("testds").
					Obj(),
				testing.MakeJSONPatchOperation().
					SetOperation("add").
					SetNewConfigMapRefasPath("containers", 0).
					AddConfigMapRefsToValue([]string{"testds"}).
					AddSecretRefsToValue([]string{"testds"}).
					Obj(),
			}
			return patchArray
		},
	}),
	Entry("Pod with no dataset volumes, 1 dataset label, useas configmap, label present in envFrom -> no patch", &testPodLabels{
		makeInputPodSpec: func() *corev1.Pod {
			inputPod := testing.MakePod("test-1", "test").
				AddLabelToPodMetadata("dataset.0.id", "testds").
				AddLabelToPodMetadata("dataset.0.useas", "configmap").
				AddContainerToPod(testing.MakeContainer("foo").
					AddEnvFromConfigToContainer("testds").
					Obj()).
				Obj()
			return &inputPod
		},
		makeOutputPatchOperations: func() []jsonpatch.JsonPatchOperation {
			return []jsonpatch.JsonPatchOperation{}
		},
	}),

	Entry("Pod with no dataset volumes, 1 dataset label, useas configmap, label present in envFrom Secret -> no patch", &testPodLabels{
		makeInputPodSpec: func() *corev1.Pod {
			inputPod := testing.MakePod("test-1", "test").
				AddLabelToPodMetadata("dataset.0.id", "testds").
				AddLabelToPodMetadata("dataset.0.useas", "configmap").
				AddContainerToPod(testing.MakeContainer("foo").
					AddEnvFromSecretToContainer("testds").
					Obj()).
				Obj()
			return &inputPod
		},
		makeOutputPatchOperations: func() []jsonpatch.JsonPatchOperation {
			return []jsonpatch.JsonPatchOperation{}
		},
	}),
)
