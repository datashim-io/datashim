package dataset

import (
	utils "github.com/IBM/dataset-lifecycle-framework/plugins/noobaa-cache-plugin/pkg"
	"context"
	comv1alpha1 "github.com/IBM/dataset-lifecycle-framework/src/dataset-operator/pkg/apis/com/v1alpha1"
	"github.com/go-logr/logr"
	"io/ioutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	//corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	//"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	//"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var log = logf.Log.WithName("controller_dataset")
const datasetFinalizer = "noobaa.finalizer.dataset.ibm.com"

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Dataset Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDataset{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("dataset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Dataset
	err = c.Watch(&source.Kind{Type: &comv1alpha1.Dataset{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Dataset
	//err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
	//	IsController: true,
	//	OwnerType:    &comv1alpha1.Dataset{},
	//})
	//if err != nil {
	//	return err
	//}

	return nil
}

// blank assignment to verify that ReconcileDataset implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDataset{}

// ReconcileDataset reconciles a Dataset object
type ReconcileDataset struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Dataset object and makes changes based on the state read
// and what is in the Dataset.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDataset) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Dataset")

	// Fetch the Dataset instance
	instance := &comv1alpha1.Dataset{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	existingDatasetInternal := &comv1alpha1.DatasetInternal{}
	err = r.client.Get(context.TODO(), request.NamespacedName, existingDatasetInternal)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("DatasetInternal doesn't exist, creating")
		} else {
			reqLogger.Info("Problem retrieving datasetInternal")
			return reconcile.Result{}, err
		}
	} else {
		reqLogger.Info("DatasetInternal exists already, no need for further processing")
		return reconcile.Result{},nil
	}

	// Check if the Memcached instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isMemcachedMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isMemcachedMarkedToBeDeleted {
		if contains(instance.GetFinalizers(), datasetFinalizer) {
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeDataset(reqLogger, instance); err != nil {
				return reconcile.Result{}, err
			}

			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			instance.SetFinalizers(remove(instance.GetFinalizers(), datasetFinalizer))
			err := r.client.Update(context.TODO(), instance)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(instance.GetFinalizers(), datasetFinalizer) {
		if err := r.addFinalizer(reqLogger, instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	params_add_external:= `{
        "name": "`+instance.ObjectMeta.Name+`",
		"endpoint_type": "S3_COMPATIBLE",
		"endpoint": "`+instance.Spec.Local["endpoint"]+`",
		"identity": "`+instance.Spec.Local["accessKeyID"]+`",
		"secret": "`+instance.Spec.Local["secretAccessKey"]+`"
		}`
	utils.MakeNoobaaRequest("account_api","add_external_connection",params_add_external)

	params_create_pool:= `{
    "name": "`+instance.ObjectMeta.Name+`",
    "connection": "`+instance.ObjectMeta.Name+`",
    "target_bucket": "`+instance.Spec.Local["bucket"]+`"
    }`
	utils.MakeNoobaaRequest("pool_api","create_namespace_resource",params_create_pool)

	params_create_bucket:= `{
		"name": "`+instance.Spec.Local["bucket"]+`-cached",
		"namespace":{
		"write_resource": "`+instance.ObjectMeta.Name+`",
		"read_resources": ["`+instance.ObjectMeta.Name+`"],
		"caching": { "ttl_ms": 6000000 }
		}
    }`
	utils.MakeNoobaaRequest("bucket_api","create_bucket",params_create_bucket)

	AccessKey, err := ioutil.ReadFile("/etc/noobaa/s3/AWS_ACCESS_KEY_ID")
	if err != nil {
		return reconcile.Result{}, err
	}

	SecretKey, err := ioutil.ReadFile("/etc/noobaa/s3/AWS_SECRET_ACCESS_KEY")
	if err != nil {
		return reconcile.Result{}, err
	}

	internalDataset := &comv1alpha1.DatasetInternal{
		ObjectMeta: metav1.ObjectMeta{
			Name:  instance.ObjectMeta.Name,
			Namespace: instance.ObjectMeta.Namespace,
			Labels: map[string]string{
				"dlf-plugin-type": "caching",
				"dlf-plugin-name": "ceph-cache-plugin",
			},
		},
		Spec: comv1alpha1.DatasetSpec{
			Local: map[string]string{
				"type": "COS",
				"accessKeyID":    string(AccessKey),
				"secretAccessKey": string(SecretKey),
				"endpoint":        "http://s3.noobaa.svc",
				"bucket":          instance.Spec.Local["bucket"]+"-cached",
			},
		},
	}
	if err := controllerutil.SetControllerReference(instance, internalDataset, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	err = r.client.Create(context.TODO(), internalDataset)
	if err != nil {
		return reconcile.Result{}, err
	}
	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists")
	return reconcile.Result{}, nil
}

func (r *ReconcileDataset) finalizeDataset(reqLogger logr.Logger, m *comv1alpha1.Dataset) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	reqLogger.Info("Finalizing started")
	params_delete_bucket:= `{
		"name": "`+m.Spec.Local["bucket"]+`-cached"
    }`
	_, err := utils.MakeNoobaaRequest("bucket_api","delete_bucket",params_delete_bucket)
	if(err != nil){
		return err
	}

	params_delete_pool:= `{
    "name": "`+m.ObjectMeta.Name+`"
    }`
	_, err = utils.MakeNoobaaRequest("pool_api","delete_namespace_resource",params_delete_pool)
	if(err != nil){
		return err
	}

	params_delete_external:= `{
        "connection_name": "`+m.ObjectMeta.Name+`"
		}`
	_, err = utils.MakeNoobaaRequest("account_api","delete_external_connection",params_delete_external)
	if(err != nil){
		return err
	}

	reqLogger.Info("Successfully finalized dataset")
	return nil
}

func (r *ReconcileDataset) addFinalizer(reqLogger logr.Logger, m *comv1alpha1.Dataset) error {
	reqLogger.Info("Adding Finalizer for the Memcached")
	m.SetFinalizers(append(m.GetFinalizers(), datasetFinalizer))

	// Update CR
	err := r.client.Update(context.TODO(), m)
	if err != nil {
		reqLogger.Error(err, "Failed to update Memcached with finalizer")
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}