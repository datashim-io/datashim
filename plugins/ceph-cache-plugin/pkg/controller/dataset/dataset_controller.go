package dataset

import (
	"context"
	"os"

	errors2 "errors"

	comv1alpha1 "github.com/IBM/dataset-lifecycle-framework/src/dataset-operator/pkg/apis/com/v1alpha1"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_dataset")

var reqLogger = log.WithValues("global", "logger")

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

const datasetsFinalizer = "hpsys.ibm.ie.com"

// Reconcile reads that state of the cluster for a Dataset object and makes changes based on the state read
// and what is in the Dataset.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDataset) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	reqLogger.Info("Reconciling Dataset")

	// Fetch the Dataset instance
	datasetInstance := &comv1alpha1.Dataset{}
	err := r.client.Get(context.TODO(), request.NamespacedName, datasetInstance)
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

	isDatasetMarkedToBeDeleted := datasetInstance.GetDeletionTimestamp() != nil
	if isDatasetMarkedToBeDeleted {
		if contains(datasetInstance.GetFinalizers(), datasetsFinalizer) {
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeDataset(reqLogger, datasetInstance); err != nil {
				return reconcile.Result{}, err
			}

			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(datasetInstance, datasetsFinalizer)
			err := r.client.Update(context.TODO(), datasetInstance)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	existingDatasetInternal := &comv1alpha1.DatasetInternal{}
	err = r.client.Get(context.TODO(), request.NamespacedName, existingDatasetInternal)
	if err == nil {
		reqLogger.Info("DatasetInternal exists, checking caching placements")
		needToUpdateStatus, err := updateCachingPlacements(r.client, request, datasetInstance, existingDatasetInternal)
		if err == nil {
			if needToUpdateStatus {
				reqLogger.Info("Setting DatasetInternal caching placements")
				err = r.client.Status().Update(context.TODO(), existingDatasetInternal)
				if err != nil {
					reqLogger.Error(err, "DatasetInternal status update error")
				}
			} else {
				reqLogger.Info("DatasetInternal caching placements already set, no need to reque")
			}
		} else {
			reqLogger.Error(err, "updateCachingPlacements error")
		}
		return reconcile.Result{}, err
	} else if !errors.IsNotFound(err) {
		//Shouldn't happen
		return reconcile.Result{}, err
	}
	reqLogger.Info("Dataset internal not ready yet, we should provision")

	clusterList := &cephv1.CephClusterList{}
	err = populateListOfObjects(r.client, clusterList, []client.ListOption{
		client.InNamespace(os.Getenv("ROOK_NAMESPACE")),
	})
	if err != nil {
		errAdd := addErrorToDataset(r.client, "cannot lookup ceph clusters", datasetInstance)
		if errAdd != nil {
			return reconcile.Result{}, errAdd
		}
		return reconcile.Result{}, err
	} else if len(clusterList.Items) == 0 {
		errAdd := addErrorToDataset(r.client, "no ceph clusters available", datasetInstance)
		if errAdd != nil {
			return reconcile.Result{}, errAdd
		}
		return reconcile.Result{}, errors.NewBadRequest("no ceph clusters available")
	}

	configMapForRados := &corev1.ConfigMap{}
	err = getExactlyOneObject(r.client, configMapForRados, "rook-ceph-rgw-"+request.Name+"-custom", os.Getenv("ROOK_NAMESPACE"))
	if errors.IsNotFound(err) {
		reqLogger.Info("errors.IsNotFound(err) object store")
		errCreation := createCustomConfigMapForRados(r.client, datasetInstance)
		if errCreation != nil {
			return reconcile.Result{}, errCreation
		}
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Info("Generic error for getting ceph object store, shouldn't happen")
		return reconcile.Result{}, err
	} else {
		reqLogger.Info("Found one, but lets check if they belog to the same dataset")
		sameObj := isSameCephObject(configMapForRados.Labels, datasetInstance)
		if sameObj == false {
			return reconcile.Result{}, errors.NewBadRequest("rgw exists, but belongs to different dataset")
		}
		reqLogger.Info("Found the correct rgw, all good")
	}

	cephObjectStore := &cephv1.CephObjectStore{}
	err = getExactlyOneObject(r.client, cephObjectStore, request.Name, os.Getenv("ROOK_NAMESPACE"))
	if errors.IsNotFound(err) {
		reqLogger.Info("errors.IsNotFound(err) object store")
		errCreation := createCephObjectStore(r.client, datasetInstance)
		if errCreation != nil {
			return reconcile.Result{}, errCreation
		}
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Info("Generic error for getting ceph object store, shouldn't happen")
		return reconcile.Result{}, err
	} else {
		reqLogger.Info("Found one, but lets check if they belog to the same dataset")
		sameObj := isSameCephObject(cephObjectStore.Labels, datasetInstance)
		if sameObj == false {
			return reconcile.Result{}, errors.NewBadRequest("rgw exists, but belongs to different dataset")
		}
		if cephObjectStore.Status == nil {
			reqLogger.Info("Rgw not ready, requeing")
			return reconcile.Result{Requeue: true}, nil
		}
		if cephObjectStore.Status != nil && cephObjectStore.Status.Phase != "Connected" {
			reqLogger.Info("Rgw not ready, requeing")
			return reconcile.Result{Requeue: true}, nil
		}
		rgwPods := &corev1.PodList{}
		err = populateListOfObjects(r.client, rgwPods, []client.ListOption{
			client.InNamespace(os.Getenv("ROOK_NAMESPACE")),
			client.MatchingLabels{"app": "rook-ceph-rgw", "rgw": request.Name},
		})
		if err != nil {
			reqLogger.Info("Error getting list of pods for rgw")
			return reconcile.Result{}, err
		} else {
			if len(rgwPods.Items) == 0 {
				reqLogger.Info("Rgw pod not ready, requeing")
				return reconcile.Result{Requeue: true}, nil
			}
		}
		reqLogger.Info("Found the correct rgw, all good")
	}

	cephObjectStoreUser := &cephv1.CephObjectStoreUser{}
	err = getExactlyOneObject(r.client, cephObjectStoreUser, request.Name, os.Getenv("ROOK_NAMESPACE"))
	if errors.IsNotFound(err) {
		reqLogger.Info("errors.IsNotFound(err) object store")
		errCreation := createCephObjectStoreUser(r.client, datasetInstance)
		if errCreation != nil {
			return reconcile.Result{}, errCreation
		}
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Info("Generic error for getting ceph object storeUser, shouldn't happen")
		return reconcile.Result{}, err
	} else {
		reqLogger.Info("Found one, but lets check if they belog to the same dataset")
		sameObj := isSameCephObject(cephObjectStoreUser.Labels, datasetInstance)
		if sameObj == false {
			return reconcile.Result{}, errors.NewBadRequest("rgw exists, but belongs to different dataset")
		}
		reqLogger.Info("Found the correct rgw, all good")
	}

	associatedCephUserSecrets := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{
		Name:      "rook-ceph-object-user-" + datasetInstance.ObjectMeta.Name + "-" + datasetInstance.ObjectMeta.Name,
		Namespace: os.Getenv("ROOK_NAMESPACE")},
		associatedCephUserSecrets)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("ceph user secrets not created yet, requeing")
		return reconcile.Result{Requeue: true}, nil
		// Secrets created successfully - don't requeue
	} else if err != nil {
		return reconcile.Result{}, err
	}
	//TODO the parent is the ceph cluster, at least in rook 1.2 so we cannot check if they are the same parent
	//else {
	//	if(len(associatedCephUserSecrets.OwnerReferences)>0 && associatedCephUserSecrets.OwnerReferences[0].UID!=cephObjectStore.UID){
	//		reqLogger.Info("CephObjectStore with different parent, requing")
	//		return reconcile.Result{Requeue: true},nil
	//	}
	//}

	associatedRgwService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{
		Name:      "rook-ceph-rgw-" + datasetInstance.ObjectMeta.Name,
		Namespace: os.Getenv("ROOK_NAMESPACE")}, associatedRgwService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("RGW service not found, requing")
		return reconcile.Result{Requeue: true}, nil
		// Secrets created successfully - don't requeue
	} else if err != nil {
		return reconcile.Result{}, err
	} else {
		if len(associatedRgwService.OwnerReferences) > 0 && associatedRgwService.OwnerReferences[0].UID != cephObjectStore.UID {
			reqLogger.Info("RgwService with different parent, requing")
			return reconcile.Result{Requeue: true}, nil
		}
	}

	AccessKey := associatedCephUserSecrets.Data["AccessKey"]
	SecretKey := associatedCephUserSecrets.Data["SecretKey"]
	InternalEndpoint := associatedRgwService.Spec.ClusterIP

	internalDatasetLabels := make(map[string]string)
	for key, value := range datasetInstance.ObjectMeta.Labels {
		internalDatasetLabels[key] = value
	}
	internalDatasetLabels["dlf-plugin-type"] = "caching"
	internalDatasetLabels["dlf-plugin-name"] = "ceph-cache-plugin"

	internalDataset := &comv1alpha1.DatasetInternal{
		ObjectMeta: metav1.ObjectMeta{
			Name:      datasetInstance.ObjectMeta.Name,
			Namespace: datasetInstance.ObjectMeta.Namespace,
			Labels:    internalDatasetLabels,
		},
		Spec: comv1alpha1.DatasetSpec{
			Local: map[string]string{
				"type":            "COS",
				"accessKeyID":     string(AccessKey),
				"secretAccessKey": string(SecretKey),
				"endpoint":        "http://" + InternalEndpoint,
				"bucket":          datasetInstance.Spec.Local["bucket"],
				"provision":       datasetInstance.Spec.Local["provision"],
				"region":          datasetInstance.Spec.Local["region"],
			},
		},
	}
	if err := controllerutil.SetControllerReference(datasetInstance, internalDataset, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	err = r.client.Create(context.TODO(), internalDataset)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !contains(datasetInstance.GetFinalizers(), datasetsFinalizer) {
		if err := r.addFinalizer(reqLogger, datasetInstance); err != nil {
			return reconcile.Result{}, err
		}
	}

	reqLogger.Info("We should stop requeuing!")

	return reconcile.Result{}, nil
}

// updateCachingPlacements checks if the dataset uses ceph-caching plugin and if caching placements
// of the datasetinternal status are set. If no action is needed it returns false, nill.
// Otherwise it returns true, nill and the caller has to issue a client status update of the datasetinternal.
// In the context of placements, updateCachingPlacements will weight the node that rgw runs on with NodeWeightRGW and
// any node that hosts one or more osds with NodeWeightOSD.
func updateCachingPlacements(rc client.Client, request reconcile.Request,
	datasetInstance *comv1alpha1.Dataset, datasetInternalInstance *comv1alpha1.DatasetInternal) (bool, error) {

	needsToWriteStatus := datasetInstance.Annotations["dlf-plugin-name"] == "ceph-cache-plugin" &&
		(len(datasetInternalInstance.Status.Caching.Placements.Gateways) == 0 ||
			len(datasetInternalInstance.Status.Caching.Placements.DataLocations) == 0)

	if needsToWriteStatus {
		//TODO assumptions:
		// * There must be at least one RGW and one OSD running pods
		// * all OSDs are coupled to a RGW

		// Query for the RGW pods aka Gateways
		gateWays := &datasetInternalInstance.Status.Caching.Placements.Gateways
		rgwPods := &corev1.PodList{}
		err := populateListOfObjects(rc, rgwPods, []client.ListOption{
			client.InNamespace(os.Getenv("ROOK_NAMESPACE")),
			client.MatchingLabels{"app": "rook-ceph-rgw", "rgw": request.Name},
		})
		if err != nil {
			reqLogger.Info("Error getting list of pods for rgw")
			return needsToWriteStatus, err
		}
		if len(rgwPods.Items) == 0 {
			errMessage := "error rgw pod not found"
			reqLogger.Info(errMessage)
			err = errors2.New(errMessage)
			return needsToWriteStatus, err
		}
		for _, rgwPod := range rgwPods.Items {
			(*gateWays) = append(*gateWays, comv1alpha1.CachingPlacementInfo{
				Key:   "kubernetes.io/hostname",
				Value: rgwPod.Spec.NodeName,
			})
		}

		// Query for the OSD pods aka DataLocations
		dataLocations := &datasetInternalInstance.Status.Caching.Placements.DataLocations
		osdPods := &corev1.PodList{}
		err = populateListOfObjects(rc, osdPods, []client.ListOption{
			client.InNamespace(os.Getenv("ROOK_NAMESPACE")),
			client.MatchingLabels{"app": "rook-ceph-osd"},
		})
		if err != nil {
			reqLogger.Info("Error getting list of ceph osds")
			return needsToWriteStatus, err
		}
		if len(osdPods.Items) == 0 {
			errMessage := "error osd pods not found"
			reqLogger.Info(errMessage)
			err = errors2.New(errMessage)
			return needsToWriteStatus, err
		}

		for _, osdPod := range osdPods.Items {
			(*dataLocations) = append(*dataLocations, comv1alpha1.CachingPlacementInfo{
				Key:   "kubernetes.io/hostname",
				Value: osdPod.Spec.NodeName,
			})
		}
	}

	return needsToWriteStatus, nil
}

func populateListOfObjects(c client.Client, listToFill interface{}, options []client.ListOption) error {

	listToFillCast, ok := listToFill.(runtime.Object)
	if !ok {
		return errors.NewBadRequest("populateListOfObjects wrong interface passed")
	}
	err := c.List(context.TODO(), listToFillCast, options...)
	if err != nil {
		return err
	}
	return nil
}

func getExactlyOneObject(c client.Client, instance runtime.Object, name string, namespace string) error {
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, instance)
	return err
}

func addErrorToDataset(c client.Client, errorString string, dataset *comv1alpha1.Dataset) error {
	dataset.Status.Caching.Info = errorString
	log.WithName("errorToDataset").Info(errorString)
	err := c.Status().Update(context.TODO(), dataset)
	if err != nil {
		return err
	}
	return nil
}
