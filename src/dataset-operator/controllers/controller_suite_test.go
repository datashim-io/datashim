package controllers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	admissioncontroller "github.com/datashim-io/datashim/src/dataset-operator/admissioncontroller"
	datasetsv1alpha1 "github.com/datashim-io/datashim/src/dataset-operator/api/v1alpha1"
	testutils "github.com/datashim-io/datashim/src/dataset-operator/testing"
	"github.com/kubernetes-csi/csi-test/v5/pkg/sanity"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	TEST_NS     = "testns"
	DATASHIM_NS = "dlf"
)

var (
	k8sClient  client.Client
	testEnv    *envtest.Environment
	cancelFunc context.CancelFunc
	ctx        context.Context
)

func TestDatasetControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Admissioncontroller Suite")
}

func initialiseMockCSIDriver() {
	_, err := testutils.NewMockCSIDriver()
	Expect(err).To(BeNil())
}

func initialiseCSISanityDriver() {
	_ = sanity.NewTestConfig()

}

func initialiseWebhookInEnvironment() {
	failedType := admissionv1.Fail
	sideEffects := admissionv1.SideEffectClassNoneOnDryRun
	reinvocationPolicy := admissionv1.IfNeededReinvocationPolicy
	webhookPath := "/mutate-pod-v1"
	timeout := int32(2)

	testEnv.WebhookInstallOptions = envtest.WebhookInstallOptions{
		LocalServingPort:    9678,
		LocalServingCertDir: "/tmp",
		MutatingWebhooks: []*admissionv1.MutatingWebhookConfiguration{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "MutatingWebhookConfiguration",
					APIVersion: "admissionregistration.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "dlf-mutating-webhook-configuration",
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name: "mpod.datashim.io",
						ClientConfig: admissionv1.WebhookClientConfig{
							URL: &webhookPath,
							Service: &admissionv1.ServiceReference{
								Name:      "webhook-server",
								Namespace: "dlf",
								Path:      &webhookPath,
							},
							CABundle: []byte{},
						},
						Rules: []admissionv1.RuleWithOperations{
							{
								Operations: []admissionv1.OperationType{"CREATE", "UPDATE"},
								Rule: admissionv1.Rule{
									APIGroups:   []string{""},
									APIVersions: []string{"v1"},
									Resources:   []string{"pods"},
								},
							},
						},
						FailurePolicy: &failedType,
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"monitor-pods-datasets": "enabled",
							},
						},
						SideEffects:             &sideEffects,
						TimeoutSeconds:          &timeout,
						ReinvocationPolicy:      &reinvocationPolicy,
						AdmissionReviewVersions: []string{"v1"},
					},
				},
			},
		},
	}
}

func initialiseRolesAndBindings(k8sClient client.Client) error {
	dssa := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{Name: "dataset-operator",
			Labels: map[string]string{
				"app.kubernetes.io/name": "dlf",
			},
			Namespace: DATASHIM_NS,
		},
	}
	err := k8sClient.Create(context.TODO(), dssa, &client.CreateOptions{})

	if err != nil {
		return err
	}

	dsRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "dataset-operator",
			Labels: map[string]string{
				"app.kubernetes.io/name": "dlf",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"datashim.io"},
				Verbs:     []string{"*"},
				Resources: []string{"*",
					"datasets", "datasetsinternal"},
				ResourceNames: []string{"dataset-operator"},
			},
			{
				APIGroups: []string{""},
				Verbs:     []string{"*"},
				Resources: []string{"pods",
					"services", "endpoints", "persistentvolumeclaims",
					"persistentvolumes", "events", "configmaps", "secrets"},
				ResourceNames: []string{"dataset-operator"},
			},
			{
				APIGroups: []string{"app"},
				Verbs:     []string{"*"},
				Resources: []string{"deployments",
					"daemonsets", "replicasets", "statefulsets"},
				ResourceNames: []string{"dataset-operator"},
			},
		},
	}

	err = k8sClient.Create(context.TODO(), dsRole, &client.CreateOptions{})
	if err != nil {
		return err
	}

	dsRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dataset-operator",
			Namespace: DATASHIM_NS,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "dataset-operator",
				Namespace: DATASHIM_NS,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "dataset-operator",
		},
	}

	err = k8sClient.Create(context.TODO(), dsRoleBinding, &client.CreateOptions{})
	if err != nil {
		return err
	} else {
		return nil
	}
}

// Test the mutation based on labels
var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancelFunc = context.WithCancel(context.Background())
	By("bootstrapping test environment")

	use_existing_cluster := true
	if os.Getenv("TEST_USE_EXISTING_CLUSTER") == "true" {
		testEnv = &envtest.Environment{
			UseExistingCluster: &use_existing_cluster,
		}
	} else {
		t := true
		testEnv = &envtest.Environment{
			CRDDirectoryPaths:     []string{filepath.Join("..", "chart", "templates", "crds")},
			BinaryAssetsDirectory: "../../../bin/k8s/1.27.1-darwin-arm64",
			ErrorIfCRDPathMissing: t,
		}
	}

	initialiseWebhookInEnvironment()

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	/*
		testEnv.CRDInstallOptions = envtest.CRDInstallOptions{
			Scheme:         &runtime.Scheme{},
			Paths:          []string{filepath.Join("..", "chart", "templates", "crds")},
			WebhookOptions: testEnv.WebhookInstallOptions,
		}

		_, err = envtest.InstallCRDs(cfg, testEnv.CRDInstallOptions)

		Expect(err).NotTo(HaveOccurred())
	*/

	/*
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: DATASHIM_NS,
			},
		}

		k8sClient.Create(context.TODO(), &ns, &client.CreateOptions{})
		Expect(err).NotTo(HaveOccurred(), "failed to create datashim namespace")
	*/
	/*
		err = initialiseRolesAndBindings(k8sClient)
		Expect(err).To(BeNil())
	*/
	/*
		err = scheme.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())
	*/
	err = datasetsv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	//+kubebuilder:scaffold:scheme
	clientset, err := kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())
	Expect(clientset).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:         scheme.Scheme,
		LeaderElection: false,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&DatasetReconciler{
		Client:    k8sManager.GetClient(),
		Scheme:    k8sManager.GetScheme(),
		Clientset: clientset,
	}).SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred(), "Could not set up Dataset Reconciler")

	err = (&DatasetInternalReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred(), "Could not set up Dataset Internal Reconciler")

	webhookServer := k8sManager.GetWebhookServer()
	webhookServer.Register("/mutate-v1-pod", &webhook.Admission{Handler: &admissioncontroller.DatasetPodMutator{}})
	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: TEST_NS,
			Labels: map[string]string{
				"monitor-pods-datasets": "enabled",
			},
		},
	}

	k8sClient.Create(context.TODO(), &ns, &client.CreateOptions{})
	Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

	//Check if we can retrieve Dataset CRDs
	crd := &apiextensionsv1.CustomResourceDefinition{}
	err = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "datasets.datashim.io"}, crd)
	Expect(err).NotTo(HaveOccurred())
	Expect(crd.Spec.Names.Kind).To(Equal("Dataset"))

	err = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "datasetsinternal.datashim.io"}, crd)
	Expect(err).NotTo(HaveOccurred())
	Expect(crd.Spec.Names.Kind).To(Equal("DatasetInternal"))
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancelFunc()

	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Test Dataset Creation", func() {
	var dataset datasetsv1alpha1.Dataset
	datasetName := "test"
	datasetNamespace := TEST_NS

	BeforeEach(func() {

		dataset = testutils.MakeDataset(datasetName, TEST_NS).ToS3Dataset("tests3", "https://", "secret", false).Obj()
		err := k8sClient.Create(context.Background(), &dataset, &client.CreateOptions{})
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("Retrieves the Dataset object", func() {

		nsName := types.NamespacedName{
			Namespace: datasetNamespace,
			Name:      datasetName,
		}

		Eventually(func() string {
			d_out := &datasetsv1alpha1.Dataset{}
			err := k8sClient.Get(context.Background(), nsName, d_out)
			Expect(err).NotTo(HaveOccurred())
			return d_out.Status.Provision.Status
		}, time.Minute, time.Second).Should(Equal(datasetsv1alpha1.StatusOK))

	})

	It("Creates a DatasetInternal object", func() {

		di := testutils.InternalFromDataset(dataset).Obj()

		nsName := types.NamespacedName{
			Namespace: datasetNamespace,
			Name:      datasetName,
		}
		di_out := &datasetsv1alpha1.DatasetInternal{}

		/*
			Eventually(func() string {
				d_out := &datasetsv1alpha1.Dataset{}
				err := k8sClient.Get(context.TODO(), nsName, d_out)
				Expect(err).NotTo(HaveOccurred())
				return d_out.Status.Provision.Status
			}, time.Minute, time.Second).Should(Equal(datasetsv1alpha1.StatusInitial))
		*/

		Eventually(func() error {
			err := k8sClient.Get(context.TODO(), nsName, di_out)
			return err
		}, time.Minute, time.Second).ShouldNot(MatchError(errors.NewNotFound(
			schema.GroupResource{Group: "datashim.io",
				Resource: "datasetsinternal"},
			"NotFound")))

		Expect(di_out).Should(BeEquivalentTo(di))

	})

	AfterEach(func() {
		err := k8sClient.Delete(context.TODO(), &dataset, &client.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred(), "failed to delete dataset")
	})
})
