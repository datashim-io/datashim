package dataset

import (
	"context"
	comv1alpha1 "github.com/IBM/dataset-lifecycle-framework/src/dataset-operator/pkg/apis/com/v1alpha1"
	connectors "github.com/YiannisGkoufas/ibm-spectrum-scale-csi/driver/csiplugin/connectors"
	"github.com/YiannisGkoufas/ibm-spectrum-scale-csi/driver/csiplugin/settings"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var log = logf.Log.WithName("controller_dataset")
const datasetFinalizer = "gpfs.finalizer.dataset.ibm.com"

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

	// Fetch the Dataset dataset
	dataset := &comv1alpha1.Dataset{}
	err := r.client.Get(context.TODO(), request.NamespacedName, dataset)
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
	isMemcachedMarkedToBeDeleted := dataset.GetDeletionTimestamp() != nil
	if isMemcachedMarkedToBeDeleted {
		if contains(dataset.GetFinalizers(), datasetFinalizer) {
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeDataset(reqLogger, dataset); err != nil {
				return reconcile.Result{}, err
			}

			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			dataset.SetFinalizers(remove(dataset.GetFinalizers(), datasetFinalizer))
			err := r.client.Update(context.TODO(), dataset)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(dataset.GetFinalizers(), datasetFinalizer) {
		if err := r.addFinalizer(reqLogger, dataset); err != nil {
			return reconcile.Result{}, err
		}
	}

	accessKeyID := dataset.Spec.Local["accessKeyID"]
	secretAccessKey := dataset.Spec.Local["secretAccessKey"]
	endpoint := dataset.Spec.Local["endpoint"]
	bucket := dataset.Spec.Local["bucket"]

	clientSpectrumScale,err := connectors.NewSpectrumRestV2(settings.Clusters{
		RestAPI: []settings.RestAPI{
			{GuiHost: "55.55.55.5", GuiPort: 443},
		},
		MgmtUsername:  "root",
		MgmtPassword:  "15121985",
	})

	if err != nil {
		return reconcile.Result{}, err
	}

	err = clientSpectrumScale.CreateBucketKeys(
		bucket,
		accessKeyID,
		secretAccessKey)
	if err != nil {
		reqLogger.Info("Error with creating keys")
		return reconcile.Result{}, err
	}
	err = clientSpectrumScale.CreateCOSFileset("scale0",dataset.ObjectMeta.Name,endpoint,bucket,true,"sw",true)
	if err != nil {
		reqLogger.Info("Error with creating COS Fileset")
		return reconcile.Result{}, err
	}

	//TODO change it to mount the exported S3FS
	internalDataset := &comv1alpha1.DatasetInternal{
		ObjectMeta: metav1.ObjectMeta{
			Name:  dataset.ObjectMeta.Name,
			Namespace: dataset.ObjectMeta.Namespace,
			Labels: map[string]string{
				"dlf-plugin-type": "caching",
				"dlf-plugin-name": "gpfs-cache-plugin",
			},
		},
		Spec: comv1alpha1.DatasetSpec{
			Local: map[string]string{
				"type": "HOST",
				"path": "/ibm/scale0/"+dataset.ObjectMeta.Name,
				"hostPathType": "Directory",
			},
		},
	}
	if err := controllerutil.SetControllerReference(dataset, internalDataset, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	err = r.client.Create(context.TODO(), internalDataset)
	if err != nil {
		return reconcile.Result{}, err
	}


	// Pod already exists - don't requeue
	//reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return reconcile.Result{}, nil
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

func (r *ReconcileDataset) finalizeDataset(reqLogger logr.Logger, m *comv1alpha1.Dataset) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.

	clientSpectrumScale,err := connectors.NewSpectrumRestV2(settings.Clusters{
		RestAPI: []settings.RestAPI{
			{GuiHost: "55.55.55.5", GuiPort: 443},
		},
		MgmtUsername:  "root",
		MgmtPassword:  "15121985",
	})
	if err != nil {
		return err
	}

	err = clientSpectrumScale.UnlinkFileset("scale0",m.ObjectMeta.Name)
	if err != nil {
		reqLogger.Info("Error when unlinking fileset")
		return err
	}

	err = clientSpectrumScale.DeleteFileset("scale0",m.ObjectMeta.Name)
	if err != nil {
		reqLogger.Info("Error when deleting fileset")
		return err
	}

	reqLogger.Info("Successfully finalized dataset")
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