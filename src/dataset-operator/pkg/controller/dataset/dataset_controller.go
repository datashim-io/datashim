package dataset

import (
	"context"
	"strconv"

	comv1alpha1 "dataset-operator/pkg/apis/com/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_dataset")

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
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &comv1alpha1.Dataset{},
	})
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

	// Fetch the Dataset instance
	instance := &comv1alpha1.Dataset{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Dataset is not found")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	reqLogger.Info("All good, proceed")

	if instance.Spec.Local != nil {
		result, err := processLocalDataset(instance, r)
		if err != nil {
			return result, err
		}
	}

	if instance.Spec.Remote != nil {
		result, err := processRemoteDataset(instance, r)
		if err != nil {
			return result, err
		}
	}

	return reconcile.Result{}, nil
}

func processLocalDataset(cr *comv1alpha1.Dataset, rc *ReconcileDataset) (reconcile.Result, error) {
	processLocalDatasetLogger := log.WithValues("Dataset.Namespace", cr.Namespace, "Dataset.Name", cr.Name, "Method", "processLocalDataset")

	accessKeyID := cr.Spec.Local["accessKeyID"]
	secretAccessKey := cr.Spec.Local["secretAccessKey"]
	endpoint := cr.Spec.Local["endpoint"]
	bucket := cr.Spec.Local["bucket"]
	region := cr.Spec.Local["region"]

	stringData := map[string]string{
		"accessKeyID":     accessKeyID,
		"secretAccessKey": secretAccessKey,
		"endpoint":        endpoint,
		"bucket":          bucket,
		"region":          region,
	}

	labels := map[string]string{
		"dataset": cr.Name,
	}

	secretObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		StringData: stringData,
	}

	if err := controllerutil.SetControllerReference(cr, secretObj, rc.scheme); err != nil {
		return reconcile.Result{}, err
	}

	found := &corev1.Secret{}
	err := rc.client.Get(context.TODO(), types.NamespacedName{Name: secretObj.Name, Namespace: secretObj.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		processLocalDatasetLogger.Info("Creating new secrets", "Secret.Namespace", secretObj.Namespace, "Secret.Name", secretObj.Name)
		err = rc.client.Create(context.TODO(), secretObj)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Secrets created successfully - don't requeue
	} else if err != nil {
		return reconcile.Result{}, err
	}

	storageClassName := "csi-s3"

	newPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("5Gi"),
				},
			},
			StorageClassName: &storageClassName,
		},
	}

	if err := controllerutil.SetControllerReference(cr, newPVC, rc.scheme); err != nil {
		return reconcile.Result{}, err
	}

	foundPVC := &corev1.PersistentVolumeClaim{}
	err = rc.client.Get(context.TODO(), types.NamespacedName{Name: newPVC.Name, Namespace: newPVC.Namespace}, foundPVC)
	if err != nil && errors.IsNotFound(err) {
		processLocalDatasetLogger.Info("Creating new pvc", "PVC.Namespace", newPVC.Namespace, "PVC.Name", newPVC.Name)
		err = rc.client.Create(context.TODO(), newPVC)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Secrets created successfully - don't requeue
	} else if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func processRemoteDataset(cr *comv1alpha1.Dataset, rc *ReconcileDataset) (reconcile.Result, error) {

	processRemoteDatasetLogger := log.WithValues("Dataset.Namespace", cr.Namespace, "Dataset.Name", cr.Name, "Method", "processRemoteDataset")

	entryType := cr.Spec.Remote["type"]
	switch entryType {
	case "CatalogEntry":
		catalogUri, ok := cr.Spec.Remote["catalogURI"]
		if !ok {
			processRemoteDatasetLogger.Error(nil, "no catalogURI provided")
			return reconcile.Result{}, errors.NewBadRequest("no catalogURI provided")
		}
		table, ok := cr.Spec.Remote["table"]
		if !ok {
			processRemoteDatasetLogger.Error(nil, "no table provided for lookup")
			return reconcile.Result{}, errors.NewBadRequest("no table provided for lookup")
		}

		var mountAllowed bool
		var err error
		mountAllowedValue, ok := cr.Spec.Remote["mountAllowed"]
		if !ok {
			processRemoteDatasetLogger.Info("No mount allowed")
			mountAllowed = false
		} else {
			mountAllowed, err = strconv.ParseBool(mountAllowedValue)
			if err != nil {
				mountAllowed = false
			}
		}

		bukits, err := processCatalogEntry(catalogUri, table)

		if err != nil {
			processRemoteDatasetLogger.Error(err, "Error in querying metastore", "catalogURI", catalogUri, "table", table)
			return reconcile.Result{}, err
		} else if len(bukits) == 0 {
			processRemoteDatasetLogger.Error(nil, "0 records obtained from the catalog ", "catalogURI", catalogUri, "table", table)
			return reconcile.Result{}, errors.NewBadRequest("No records obtained from the catalog")
		}

		bucketData := make(map[string]string)
		bucketData["catalogURI"] = catalogUri
		bucketData["table"] = table
		bucketData["numBuckets"] = strconv.Itoa(len(bukits))
		for i, bkt := range bukits {
			bucketData["bucket."+strconv.Itoa(i)] = bkt
		}

		labels := map[string]string{
			"dataset": cr.Name,
		}

		configMapObject := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cr.Name,
				Namespace: cr.Namespace,
				Labels:    labels,
			},
			Data: bucketData,
		}

		if err := controllerutil.SetControllerReference(cr, configMapObject, rc.scheme); err != nil {
			return reconcile.Result{}, err
		}

		foundConfigMap := &corev1.ConfigMap{}
		err = rc.client.Get(context.TODO(), types.NamespacedName{Name: configMapObject.Name, Namespace: configMapObject.Namespace}, foundConfigMap)

		if err != nil && errors.IsNotFound(err) {
			processRemoteDatasetLogger.Info("Creating new configMap", "configMap.namespace",
				configMapObject.Namespace, "PVC.Name", configMapObject.Name)
			err = rc.client.Create(context.TODO(), configMapObject)
			if err != nil {
				return reconcile.Result{}, err
			}
			// configMap created successfully - don't requeue
		} else if err != nil {
			return reconcile.Result{}, err
		}

		//Supporting only a single location for the time being
		if mountAllowed {
			processRemoteDatasetLogger.Info("Creating a PVC for a single bucket", "bucketName", bucketData["bucket.0"])

			storageClassName := "csi-s3"

			processRemoteDatasetLogger.Info("Creating a Secret for a single bucket", "bucketName", bucketData["bucket.0"])

			if _, err := createSecretForBucket(bucketData["bucket.0"], cr, rc); err != nil {
				processRemoteDatasetLogger.Error(err, "Creating Secret failed", "bucketName", bucketData["bucket.0"])
				return reconcile.Result{}, errors.NewServiceUnavailable("Unable to initialise dataset")
			}

			newPVC := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cr.Name,
					Namespace: cr.Namespace,
					Labels:    labels,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("5Gi"),
						},
					},
					StorageClassName: &storageClassName,
				},
			}

			if err := controllerutil.SetControllerReference(cr, newPVC, rc.scheme); err != nil {
				return reconcile.Result{}, err
			}

			foundPVC := &corev1.PersistentVolumeClaim{}
			err = rc.client.Get(context.TODO(), types.NamespacedName{Name: newPVC.Name, Namespace: newPVC.Namespace}, foundPVC)
			if err != nil && errors.IsNotFound(err) {
				processRemoteDatasetLogger.Info("Creating new pvc", "PVC.Namespace", newPVC.Namespace, "PVC.Name", newPVC.Name)
				err = rc.client.Create(context.TODO(), newPVC)
				if err != nil {
					return reconcile.Result{}, err
				}
				// Secrets created successfully - don't requeue
			} else if err != nil {
				return reconcile.Result{}, err
			}
		}

	default:
		err := errors.NewBadRequest("Unsupported dataset entry type")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func createSecretForBucket(bucket string, cr *comv1alpha1.Dataset, rc *ReconcileDataset) (reconcile.Result, error) {

	createSecretForBucketLogger := log.WithValues("Dataset.Namespace", cr.Namespace, "Dataset.Name", cr.Name, "Method", "createSecretForBucket")

	accessKeyID := cr.Spec.Remote["accessKeyID"]
	secretAccessKey := cr.Spec.Remote["secretAccessKey"]
	endpoint := cr.Spec.Remote["endpoint"]
	region := cr.Spec.Remote["region"]

	stringData := map[string]string{
		"accessKeyID":     accessKeyID,
		"secretAccessKey": secretAccessKey,
		"endpoint":        endpoint,
		"bucket":          bucket,
		"region":          region,
	}

	labels := map[string]string{
		"dataset": cr.Name,
	}

	secretObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		StringData: stringData,
	}

	if err := controllerutil.SetControllerReference(cr, secretObj, rc.scheme); err != nil {
		createSecretForBucketLogger.Error(err, "Could not set secret object for dataset", "name", cr.Name)
		return reconcile.Result{}, err
	}

	found := &corev1.Secret{}
	err := rc.client.Get(context.TODO(), types.NamespacedName{Name: secretObj.Name, Namespace: secretObj.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		createSecretForBucketLogger.Info("Creating new secrets", "Secret.Namespace", secretObj.Namespace, "Secret.Name", secretObj.Name)
		err = rc.client.Create(context.TODO(), secretObj)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Secrets created successfully - don't requeue
	} else if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
