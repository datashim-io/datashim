package datasetinternal

import (
	"context"
	"strconv"

	comv1alpha1 "github.com/IBM/dataset-lifecycle-framework/src/dataset-operator/pkg/apis/com/v1alpha1"
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

var log = logf.Log.WithName("controller_datasetinternal")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

/**
 * Functions table for hadnling creation of local datasets.
 * Each function in the table should respect the following signature:
 *		processLocalDatasetXYZ func(*comv1alpha1.Dataset, *ReconcileDataset) (reconcile.Result, error)
 */
var datasetLocalProcessTable = map[string]func(*comv1alpha1.DatasetInternal,
	*ReconcileDatasetInternal) (reconcile.Result, error){
	"COS":  processLocalDatasetCOS,
	"NFS":  processLocalDatasetNFS,
	"HOST": processLocalDatasetHOST,
}

// Add creates a new Dataset Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDatasetInternal{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("dataset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Dataset
	err = c.Watch(&source.Kind{Type: &comv1alpha1.DatasetInternal{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Dataset
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &comv1alpha1.DatasetInternal{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &comv1alpha1.DatasetInternal{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.PersistentVolumeClaim{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &comv1alpha1.DatasetInternal{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileDatasetInternal implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDatasetInternal{}

// ReconcileDataset reconciles a Dataset object
type ReconcileDatasetInternal struct {
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
func (r *ReconcileDatasetInternal) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Dataset")

	result := reconcile.Result{}
	var err error = nil

	// Fetch the Dataset instance
	instance := &comv1alpha1.DatasetInternal{}
	err = r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Dataset is not found")
			err = nil
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	reqLogger.Info("All good, proceed")

	if instance.Spec.Local != nil {
		datasetType := instance.Spec.Local["type"]
		if f, ok := datasetLocalProcessTable[datasetType]; ok {
			result, err = f(instance, r)
		} else {
			reqLogger.Error(err, "Dataset type %s not supported", datasetType)
			err = errors.NewBadRequest("Dataset type not supported")
		}
	} else if instance.Spec.Remote != nil {
		result, err = processRemoteDataset(instance, r)
		if err != nil {
			reqLogger.Error(err, "Could not process remote dataset entry: %v", instance.Name)
			err = errors.NewBadRequest("Could not process remote dataset")
		}
	} else {
		reqLogger.Info("Unknown spec entry")
		err = errors.NewBadRequest("Dataset type not supported")
	}

	return result, err
}

func processLocalDatasetCOS(cr *comv1alpha1.DatasetInternal, rc *ReconcileDatasetInternal) (reconcile.Result, error) {
	processLocalDatasetLogger := log.WithValues("Dataset.Namespace", cr.Namespace, "Dataset.Name", cr.Name, "Method", "processLocalDataset")

	accessKeyID := cr.Spec.Local["accessKeyID"]
	secretAccessKey := cr.Spec.Local["secretAccessKey"]
	endpoint := cr.Spec.Local["endpoint"]
	bucket := cr.Spec.Local["bucket"]
	region := cr.Spec.Local["region"]
	readonly := "false"
	if readonlyValueString, ok := cr.Spec.Local["readonly"]; ok {
		readonly = readonlyValueString
	}

	stringData := map[string]string{
		"accessKeyID":     accessKeyID,
		"secretAccessKey": secretAccessKey,
		"endpoint":        endpoint,
		"bucket":          bucket,
		"region":          region,
		"readonly":        readonly,
	}

	labels := map[string]string{
		"dataset": cr.Name,
	}

	configData := map[string]string{
		"endpoint": endpoint,
		"bucket":   bucket,
		"region":   region,
	}

	if _, err := createConfigMapforDataset(configData, cr, rc); err != nil {
		processLocalDatasetLogger.Error(err, "Could not create ConfigMap for dataset", "Dataset.Name", cr.Name)
		return reconcile.Result{}, err
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
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
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

func processLocalDatasetNFS(cr *comv1alpha1.DatasetInternal, rc *ReconcileDatasetInternal) (reconcile.Result, error) {
	processLocalDatasetLogger := log.WithValues("Dataset.Namespace", cr.Namespace, "Dataset.Name", cr.Name, "Method", "processLocalDatasetNFS")
	processLocalDatasetLogger.Info("Dataset type NFS")

	server := cr.Spec.Local["server"]
	share := cr.Spec.Local["share"]

	labels := map[string]string{
		"dataset": cr.Name,
	}

	storageClassName := "csi-nfs"
	csiDriverName := "csi-nfsplugin"
	csiVolumeHandle := "data-id"
	csiVolumeAttributes := map[string]string{
		"server": server,
		"share":  share,
	}
	pvSource := &corev1.CSIPersistentVolumeSource{
		Driver:           csiDriverName,
		VolumeHandle:     csiVolumeHandle,
		VolumeAttributes: csiVolumeAttributes,
	}

	newPV := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeSpec{
			StorageClassName: storageClassName,
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("5Gi"), //TODO: use proper size
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: pvSource,
			},
		},
	}

	if err := controllerutil.SetControllerReference(cr, newPV, rc.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// the csi-nfs plugin does not support dynamic provisioning so PV and PVC must be created manually
	foundPV := &corev1.PersistentVolume{}
	err := rc.client.Get(context.TODO(), types.NamespacedName{Name: newPV.Name, Namespace: newPV.Namespace}, foundPV)
	if err != nil && errors.IsNotFound(err) {
		processLocalDatasetLogger.Info("Creating new PV", "PV.Namespace", newPV.Namespace, "PV.Name", newPV.Name)
		err = rc.client.Create(context.TODO(), newPV)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}

	newPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("5Gi"), //TODO: use proper size
				},
			},
			StorageClassName: &storageClassName,
			VolumeName:       cr.Name,
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
	}
	return reconcile.Result{}, nil
}

/* This function creates a dataset out of an already existing path in the host.
 * HostPath is well supported in Kubernetes so I am not sure we really need the
 * complexity of doing this via the hostpath CSI driver:
 *    https://github.com/kubernetes-csi/csi-driver-host-path
 */
func processLocalDatasetHOST(cr *comv1alpha1.DatasetInternal, rc *ReconcileDatasetInternal) (reconcile.Result, error) {
	processLocalDatasetLogger := log.WithValues("Dataset.Namespace", cr.Namespace, "Dataset.Name", cr.Name, "Method", "processLocalDatasetHOST")
	processLocalDatasetLogger.Info("Dataset type HOST")

	hostpathType := corev1.HostPathDirectory
	permissions := corev1.ReadWriteMany
	path := ""
	storageClassName := "hostpath"
	/* the 'path' field is mandatory and must always contain a non empty string.
	 * I am assuming the validity of its value is checked sometime later when
	 * the PVC is bound to a pod.
	 */
	if path, _ = cr.Spec.Local["path"]; len(path) == 0 {
		err := errors.NewBadRequest("path missing or not valid")
		processLocalDatasetLogger.Error(err, "path missing or not valid")
		return reconcile.Result{}, err
	}

	/* We support the following permission schemes:
	 * - ReadOnly
	 * - ReadWrite
	 * NOTE: As of today creating a read-only PV and PVC does not make it
	 * read-only at POD runtime. AccessMode is only used at PVC-PV binding time
	 * to make sure it is a valid binding but it is not transferred to the POD.
	 * We'll have to handle this while mutating the PODs.
	 */
	if perm, exists := cr.Spec.Local["permissions"]; exists {
		switch perm {
		case "ReadOnly":
			permissions = corev1.ReadOnlyMany
		case "", "ReadWrite":
			permissions = corev1.ReadWriteMany
		default:
			err := errors.NewBadRequest("permissions not supported")
			processLocalDatasetLogger.Error(err, "permissions '%s' not supported", perm)
			return reconcile.Result{}, err
		}
	}

	/* We support the following types:
	 * - Directory: path must exist at dataset creation time (Default value)
	 * - CreateNew: path is created if not existing
	 */
	if hostpath_type_string, exists := cr.Spec.Local["hostPathType"]; exists {
		switch hostpath_type_string {
		case "CreateNew":
			hostpathType = corev1.HostPathDirectoryOrCreate
		case "", "Directory":
			hostpathType = corev1.HostPathDirectory
		default:
			err := errors.NewBadRequest("HostPath type not supported")
			processLocalDatasetLogger.Error(err, "HostPath type %s not supported", hostpath_type_string)
			return reconcile.Result{}, err
		}
	}

	labels := map[string]string{
		"dataset": cr.Name,
	}

	pvSource := &corev1.HostPathVolumeSource{
		Path: path,
		Type: &hostpathType,
	}

	newPV := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeSpec{
			StorageClassName: storageClassName,
			AccessModes:      []corev1.PersistentVolumeAccessMode{permissions},
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("5Gi"),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: pvSource,
			},
		},
	}

	if err := controllerutil.SetControllerReference(cr, newPV, rc.scheme); err != nil {
		return reconcile.Result{}, err
	}

	foundPV := &corev1.PersistentVolume{}
	err := rc.client.Get(context.TODO(), types.NamespacedName{Name: newPV.Name, Namespace: newPV.Namespace}, foundPV)
	if err != nil && errors.IsNotFound(err) {
		processLocalDatasetLogger.Info("Creating new PV", "PV.Namespace", newPV.Namespace, "PV.Name", newPV.Name)
		err = rc.client.Create(context.TODO(), newPV)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}

	newPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{permissions},
			VolumeName:       cr.Name,
			StorageClassName: &storageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("5Gi"), //TODO: use proper size
				},
			},
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
	}
	return reconcile.Result{}, nil

}

func processRemoteDataset(cr *comv1alpha1.DatasetInternal, rc *ReconcileDatasetInternal) (reconcile.Result, error) {

	processRemoteDatasetLogger := log.WithValues("Dataset.Namespace", cr.Namespace, "Dataset.Name", cr.Name, "Method", "processRemoteDataset")
	result := reconcile.Result{}
	var err error = nil

	entryType := cr.Spec.Remote["type"]
	switch entryType {
	case "CatalogEntry":
		var catalogHost string
		var catalogPort int

		catalogUri, ok := cr.Spec.Remote["catalogURI"]
		if !ok {
			processRemoteDatasetLogger.Info("no catalogURI provided in dataset spec.. now looking up cluster services", "Dataset.Name", cr.Name)
			// Looking hivemetastore service endpoint in the cluster
			//svcList := &corev1.ServiceList{}

			// Note: Below logic searches for a particular service implementation which requires a Fieldindexer to be
			// initialised along with the controller. To make it easier, we are going to directly get the service
			// endpoind using types.NamespacedName
			//processRemoteDatasetLogger.Info("CatalogURI not provided, looking up catalog endpoint in the cluster")
			//svcListOpts := client.MatchingField("metadata.name", "hivemetastore")
			//svcListOpts = svcListOpts.InNamespace(cr.Namespace)
			//err := rc.client.List(context.TODO(), svcListOpts, svcList)

			//We are only looking for the hivemetastore endpoint in the same namespace as the dataset being created
			//TODO: Change this to be across all namespaces
			catalogSvcName := "hivemetastore"
			catalogSvcNamespace := cr.Namespace
			svc := &corev1.Service{}
			err = rc.client.Get(context.TODO(), types.NamespacedName{Name: catalogSvcName, Namespace: catalogSvcNamespace}, svc)

			if err != nil {
				processRemoteDatasetLogger.Error(err, "Could not obtain any catalogs in the current cluster")
				err = errors.NewBadRequest("no catalogURI provided")
				return result, err
			} else {
				processRemoteDatasetLogger.Info("Endpoint", "name", svc)
				catalogHost = svc.Spec.ClusterIP
				for _, port := range svc.Spec.Ports {
					processRemoteDatasetLogger.Info("Port", "name", svc)
					if port.Name == "metastore" ||
						port.Name == "" {
						catalogPort = int(port.Port)
					}
				}

				if catalogHost == "" {
					processRemoteDatasetLogger.Error(nil, "no catalogURI provided.. cannot instantiate dataset")
					err = errors.NewBadRequest("Catalog address was not found")
					return result, err
				}
			}
		} else {
			catalogHost, catalogPort, err = parseCatalogUri(catalogUri)
			if err != nil {
				processRemoteDatasetLogger.Error(err, "Could not parse CatalogUri", "catalogURI", catalogUri)
				err = errors.NewBadRequest("CatalogUri in the wrong format")
				return result, err
			}
		}
		//TODO: We expect this to change
		table, ok := cr.Spec.Remote["table"]
		if !ok {
			processRemoteDatasetLogger.Error(nil, "no table provided for lookup")
			err = errors.NewBadRequest("no table provided for lookup")
			return result, err
		}
		//TODO: We expect that we'll get the endpoints differently
		endpoint, ok := cr.Spec.Remote["endpoint"]
		if !ok {
			processRemoteDatasetLogger.Error(nil, "no endpoints provided for s3 buckets")
			err = errors.NewBadRequest("no endpoints provided for s3 buckets")
			return result, err
		}

		var mountAllowed bool

		mountAllowedValue, ok := cr.Spec.Remote["mountAllowed"]
		if !ok {
			processRemoteDatasetLogger.Info("No mount allowed")
			mountAllowed = false
		} else {
			var parseError error
			mountAllowed, parseError = strconv.ParseBool(mountAllowedValue)
			if parseError != nil {
				mountAllowed = false
			}
		}

		var bukits []string
		bukits, err = processCatalogEntry(catalogHost, int(catalogPort), table)

		if err != nil {
			processRemoteDatasetLogger.Error(err, "Error in querying metastore", "catalogURI", catalogUri, "table", table)
			err = errors.NewBadRequest("Could not query catalog at address: " + catalogUri)
			return result, err
		} else if len(bukits) == 0 {
			processRemoteDatasetLogger.Error(nil, "0 records obtained from the catalog ", "catalogURI", catalogUri, "table", table)
			err = errors.NewBadRequest("No records obtained from the catalog")
			return result, err
		}

		bucketData := make(map[string]string)
		for i, bkt := range bukits {
			bucketData["bucket."+strconv.Itoa(i)] = bkt
		}

		// Create the config map anyway as we do not know if a pod will mount the bucket or opt for environment injection

		processRemoteDatasetLogger.Info("Creating a ConfigMap for the bucket data obtained for dataset", "datasetName", cr.Name, "tableName", table)
		bucketData["catalogHost"] = catalogHost
		bucketData["catalogPort"] = strconv.FormatInt(int64(catalogPort), 10)
		bucketData["table"] = table
		bucketData["numBuckets"] = strconv.Itoa(len(bukits))

		if _, err := createConfigMapforDataset(bucketData, cr, rc); err != nil {
			processRemoteDatasetLogger.Error(err, "Could not create ConfigMap for dataset", "dataset", cr.Name)
			return reconcile.Result{}, errors.NewServiceUnavailable("Unable to initialise dataset")
		}

		processRemoteDatasetLogger.Info("Creating a Secret for the bucket data obtain for table", "tableName", table)
		labels := map[string]string{
			"dataset": cr.Name,
		}
		// We are creating this as a secret as it contains the access key and secret access key for Obj. Storage access
		// We need this secret for both mounting and for environment variable injection
		secretData := make(map[string]string)
		secretData["endpoint"] = endpoint
		secretData["accessKeyID"] = cr.Spec.Remote["accessKeyID"]
		secretData["secretAccessKey"] = cr.Spec.Remote["secretAccessKey"]
		secretData["region"] = cr.Spec.Remote["region"]
		// To reduce duplication, we are entering the bucket information in this secret. This is so that csi-s3
		// can mount the right bucket in the pod. Be aware that this secret needs to be created for each
		// bucket that has to be mounted in the pod.
		// Currently, we are only supporting mounting a single bucket inside the pod.
		secretData["bucket"] = bucketData["bucket.0"]

		secretObj := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cr.Name,
				Namespace: cr.Namespace,
				Labels:    labels,
			},
			StringData: secretData,
		}

		if err := controllerutil.SetControllerReference(cr, secretObj, rc.scheme); err != nil {
			processRemoteDatasetLogger.Error(err, "Could not set secret object for dataset", "name", cr.Name)
			return reconcile.Result{}, err
		}

		found := &corev1.Secret{}
		err := rc.client.Get(context.TODO(), types.NamespacedName{Name: secretObj.Name, Namespace: secretObj.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			processRemoteDatasetLogger.Info("Creating new secrets", "Secret.Namespace", secretObj.Namespace, "Secret.Name", secretObj.Name)
			err = rc.client.Create(context.TODO(), secretObj)
			if err != nil {
				return reconcile.Result{}, err
			}
			// Secrets created successfully - don't requeue
		} else if err != nil {
			return reconcile.Result{}, err
		}

		//Supporting only a single location for the time being
		if mountAllowed {
			processRemoteDatasetLogger.Info("Creating a PVC for a single bucket", "bucketName", bucketData["bucket.0"])

			if _, err := createPVCforObjectStorage(cr, rc); err != nil {
				processRemoteDatasetLogger.Error(err, "Mounting of object storage bucket failed", "bucketName", bucketData["bucket.0"])
				return reconcile.Result{}, errors.NewServiceUnavailable("Unable to initialise dataset")
			}

		}

	default:
		err := errors.NewBadRequest("Unsupported dataset entry type")
		return result, err
	}

	return result, nil
}

func createConfigMapforDataset(configMapData map[string]string, cr *comv1alpha1.DatasetInternal, rc *ReconcileDatasetInternal) (reconcile.Result, error) {

	createConfigMapLogger := log.WithValues("Dataset.Namespace", cr.Namespace, "Dataset.Name", cr.Name, "Method", "createConfigMapforObjectStorage")
	result := reconcile.Result{}
	var err error = nil

	labels := map[string]string{
		"dataset": cr.Name,
	}

	configMapObject := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Data: configMapData,
	}

	if err = controllerutil.SetControllerReference(cr, configMapObject, rc.scheme); err != nil {
		return result, err
	}

	foundConfigMap := &corev1.ConfigMap{}
	err = rc.client.Get(context.TODO(), types.NamespacedName{Name: configMapObject.Name, Namespace: configMapObject.Namespace}, foundConfigMap)

	if err != nil && errors.IsNotFound(err) {
		createConfigMapLogger.Info("Creating new configMap", "configMap.namespace",
			configMapObject.Namespace, "configMap.Name", configMapObject.Name)
		err = rc.client.Create(context.TODO(), configMapObject)
		if err != nil {
			return result, err
		}
	} else if err != nil {
		return result, err
	}

	return result, err
}

func createPVCforObjectStorage(cr *comv1alpha1.DatasetInternal, rc *ReconcileDatasetInternal) (reconcile.Result, error) {

	createPVCLogger := log.WithValues("Dataset.Namespace", cr.Namespace, "Dataset.Name", cr.Name, "Method", "createPVCforObjectStorage")
	result := reconcile.Result{}
	var err error = nil

	storageClassName := "csi-s3"

	labels := map[string]string{
		"dataset": cr.Name,
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

	if err = controllerutil.SetControllerReference(cr, newPVC, rc.scheme); err == nil {

		foundPVC := &corev1.PersistentVolumeClaim{}
		err = rc.client.Get(context.TODO(), types.NamespacedName{Name: newPVC.Name, Namespace: newPVC.Namespace}, foundPVC)
		if err != nil && errors.IsNotFound(err) {
			//PVC not created - requeue
			createPVCLogger.Info("Creating new pvc", "PVC.Namespace", newPVC.Namespace, "PVC.Name", newPVC.Name)
			err = rc.client.Create(context.TODO(), newPVC)
		}
	}

	return result, err

}
