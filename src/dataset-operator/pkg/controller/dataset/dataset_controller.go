package dataset

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	comv1alpha1 "github.com/IBM/dataset-lifecycle-framework/src/dataset-operator/pkg/apis/com/v1alpha1"
	"github.com/go-logr/logr"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	reconciler, err := newReconciler(mgr)
	if err != nil {
		return err
	}

	return add(mgr, reconciler)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {

	clientSet, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	return &ReconcileDataset{
		clientSet: clientSet,
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
	}, nil
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
	err = c.Watch(&source.Kind{Type: &comv1alpha1.DatasetInternal{}}, &handler.EnqueueRequestForOwner{
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
	// This clientSet reads and writes directly to the apiserver
	// and it is necessary for reading events as the above split client
	// does not support this. Note that this clientSet does not support CRDs
	clientSet *kubernetes.Clientset
	scheme    *runtime.Scheme
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

	datasetInstance := &comv1alpha1.Dataset{}
	err := r.client.Get(context.TODO(), request.NamespacedName, datasetInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Dataset is deleted", "name", request.NamespacedName)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Make sure that the Status of the Dataset is initialised
	if initializeDatasetStatus(datasetInstance) {
		return updateDatasetStatusAndReturn(r.client, datasetInstance, reqLogger)
	}

	// Caching setup
	if datasetInstance.Status.Caching.Status == comv1alpha1.StatusInitial {

		cacheDisableLabel, err := strconv.ParseBool(datasetInstance.Labels["cache.disable"])

		if err != nil {
			cacheDisableLabel = false
		}

		if cacheDisableLabel {
			// mark caching status as disabled and requeue
			datasetInstance.Status.Caching.Status = comv1alpha1.StatusDisabled
			datasetInstance.Status.Caching.Info = "User explicitly disabled caching"
			return updateDatasetStatusAndReturn(r.client, datasetInstance, reqLogger)
		}

		// get the installed DLF caching plugins
		pluginPods, err := getCachingPlugins(r.client)
		if err != nil {
			return reconcile.Result{}, err
		}

		// This means that we should create a 1-1 DatasetInternal
		if len(pluginPods.Items) != 0 {
			cachingPluginName, annotationsAreSet := datasetHasCachingAnnotationsSet(datasetInstance)
			if !annotationsAreSet {
				// if dataset does not contain the proper dlf caching annotations
				// we must add them
				cachePluginLabel, foundCachePluginLabel := datasetInstance.Labels["cache.plugin"]

				if !foundCachePluginLabel {
					//Default behavior: No cache.plugin label specified in the dataset
					//thus pick the first plugin for the time being and requeue
					datasetInstance.Annotations = pluginPods.Items[0].Labels
					return updateDatasetAndReturn(r.client, datasetInstance, reqLogger)
				}

				//User has specified the cache.plugin label
				for _, pluginItems := range pluginPods.Items {
					if pluginItems.Labels["dlf-plugin-name"] == cachePluginLabel {
						// Found the user specified plugin. Insert the proper annotations
						// and requeue
						datasetInstance.Annotations = pluginItems.Labels
						return updateDatasetAndReturn(r.client, datasetInstance, reqLogger)
					}
				}

				// User specified plugin not found. Disable caching and requeue
				datasetInstance.Status.Caching.Status = comv1alpha1.StatusDisabled
				datasetInstance.Status.Caching.Info = fmt.Sprintf(
					"User specified plugin '%s' was not found. Falling back to disabled caching",
					cachePluginLabel)
				return updateDatasetStatusAndReturn(r.client, datasetInstance, reqLogger)

			}

			// Dataset has the caching annotation embedded so now update accordingly
			// the status and reconcile
			// TODO the caching plugin is responsible for updating the caching status to OK
			datasetInstance.Status.Caching.Status = comv1alpha1.StatusPending
			datasetInstance.Status.Caching.Info = fmt.Sprintf(
				"Caching is assigned to %s plugin",
				cachingPluginName)
			return updateDatasetStatusAndReturn(r.client, datasetInstance, reqLogger)
		}

		datasetInstance.Status.Caching.Status = comv1alpha1.StatusDisabled
		datasetInstance.Status.Caching.Info = "No DLF caching plugins are installed"
		return updateDatasetStatusAndReturn(r.client, datasetInstance, reqLogger)
	}

	// We need to create DatasetInternal when caching is disabled for this Dataset
	// otherwise DatasetInternal is handled/created by the caching-plugin
	if datasetInstance.Status.Caching.Status == comv1alpha1.StatusDisabled {
		datasetInternalInstance := &comv1alpha1.DatasetInternal{}
		err = r.client.Get(context.TODO(), request.NamespacedName, datasetInternalInstance)
		if err != nil && !errors.IsNotFound(err) {
			//Unknown error occured, shouldn't happen
			return reconcile.Result{}, err
		} else if err != nil && errors.IsNotFound(err) {
			//1-1 Dataset and DatasetInternal because there is no caching plugin
			reqLogger.Info("1-1 Dataset and DatasetInternal because there is no caching plugin")

			newDatasetInternalInstance := &comv1alpha1.DatasetInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      datasetInstance.ObjectMeta.Name,
					Namespace: datasetInstance.ObjectMeta.Namespace,
				},
				Spec: datasetInstance.Spec,
			}

			if len(datasetInstance.Spec.Type) > 0 && datasetInstance.Spec.Type == "ARCHIVE" {
				podDownloadJob, bucket := getPodDataDownload(datasetInstance, os.Getenv("OPERATOR_NAMESPACE"))
				err = r.client.Create(context.TODO(), podDownloadJob)
				if err != nil {
					reqLogger.Error(err, "Error while creating pod download")
					return reconcile.Result{}, err
				}
				minioConf := &v1.Secret{}
				err = r.client.Get(context.TODO(), types.NamespacedName{
					Namespace: os.Getenv("OPERATOR_NAMESPACE"),
					Name:      "minio-conf",
				}, minioConf)
				if err != nil {
					reqLogger.Error(err, "Error while getting minio-conf secret")
					return reconcile.Result{}, err
				}
				endpoint, _ := b64.StdEncoding.DecodeString(b64.StdEncoding.EncodeToString(minioConf.Data["ENDPOINT"]))
				accessKey, _ := b64.StdEncoding.DecodeString(b64.StdEncoding.EncodeToString(minioConf.Data["AWS_ACCESS_KEY_ID"]))
				secretAccessKey, _ := b64.StdEncoding.DecodeString(b64.StdEncoding.EncodeToString(minioConf.Data["AWS_SECRET_ACCESS_KEY"]))
				reqLogger.Info(string(endpoint))

				provision := "false"

				if provisionValueString, ok := datasetInstance.Spec.Local["provision"]; ok {
					provisionBool, err := strconv.ParseBool(provisionValueString)
					if err == nil {
						provision = strconv.FormatBool(provisionBool)
					}
				}

				extract := "false"
				if len(datasetInstance.Spec.Extract) > 0 {
					extract = datasetInstance.Spec.Extract
				}
				newDatasetInternalInstance.Spec = comv1alpha1.DatasetSpec{
					Local: map[string]string{
						"type":            "COS",
						"accessKeyID":     string(accessKey),
						"secretAccessKey": string(secretAccessKey),
						"endpoint":        string(endpoint),
						"readonly":        "true",
						"bucket":          bucket,
						"extract":         extract,
						"region":          "",
						"provision":       provision,
					},
				}
			}

			if err := controllerutil.SetControllerReference(datasetInstance, newDatasetInternalInstance, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			err = r.client.Create(context.TODO(), newDatasetInternalInstance)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Provision Status
	// TODO for now for every Dataset kind we create a PVC, so the following is OK
	// However, we should make this check for PVC status only if the Dataset kind
	// is coupled with a PVC
	if datasetInstance.Status.Provision.Status != comv1alpha1.StatusOK {
		foundPVC := &v1.PersistentVolumeClaim{}
		err = r.client.Get(context.TODO(), request.NamespacedName, foundPVC)
		if err != nil {
			if errors.IsNotFound(err) {
				// If PVC is not found yet, reconcile with some delay
				if datasetInstance.Status.Provision.Status != comv1alpha1.StatusPending {
					datasetInstance.Status.Provision.Status = comv1alpha1.StatusPending
					return updateDatasetStatusAndReturn(r.client, datasetInstance, reqLogger)
				}

				reqLogger.Info("PVC is not created yet")
				return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
			}
			reqLogger.Error(err, "Unknown error")
			return reconcile.Result{}, err
		}

		switch foundPVC.Status.Phase {
		case v1.ClaimLost:
		case v1.ClaimPending:
			// ClaimLost & ClaimPending means that something is wrong with our PVC
			// lets check for relevant events
			statusUpdated := false
			if datasetInstance.Status.Provision.Status != comv1alpha1.StatusPending {
				datasetInstance.Status.Provision.Status = comv1alpha1.StatusPending
				statusUpdated = true
			}
			lastEvent, err := readEventsForPVC(r.clientSet, foundPVC, reqLogger)
			if err == nil && datasetInstance.Status.Provision.Info != lastEvent {
				datasetInstance.Status.Provision.Info = lastEvent
				statusUpdated = true
			}

			if statusUpdated {
				return updateDatasetStatusAndReturn(r.client, datasetInstance, reqLogger)
			}
			// In the case the Dataset Status is not modified requeue with some delay
			// to re-check the status
			return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil

		case v1.ClaimBound:
			// if our PVC is bound make sure that the Provision Status of Dataset is
			// shown as OK and requeue
			if datasetInstance.Status.Provision.Status != comv1alpha1.StatusOK {
				datasetInstance.Status.Provision.Status = comv1alpha1.StatusOK
				datasetInstance.Status.Provision.Info = ""
				return updateDatasetStatusAndReturn(r.client, datasetInstance, reqLogger)
			}
		}
	}

	// everything OK don't requeue
	// TODO maybe requeue with some delay if we
	// need to monitor statuses
	return reconcile.Result{}, nil
}

// initializeDataset initializing the Status of Dataset and
// returns true if it modified the Status of the Dataset or
// false otherwise
func initializeDatasetStatus(d *comv1alpha1.Dataset) bool {
	modifiedDatasetStatus := false
	if d.Status.Provision.Status == comv1alpha1.StatusEmpty {
		d.Status.Provision.Status = comv1alpha1.StatusInitial
		modifiedDatasetStatus = true
	}
	if d.Status.Caching.Status == comv1alpha1.StatusEmpty {
		d.Status.Caching.Status = comv1alpha1.StatusInitial
		modifiedDatasetStatus = true
	}
	return modifiedDatasetStatus
}

func updateDatasetStatusAndReturn(c client.Client, d *comv1alpha1.Dataset, logger logr.Logger) (reconcile.Result, error) {
	err := c.Status().Update(context.TODO(), d)
	if err != nil {
		logger.Error(err, "Error updating dataset status")
	}
	return reconcile.Result{}, err
}

func updateDatasetAndReturn(c client.Client, d *comv1alpha1.Dataset, logger logr.Logger) (reconcile.Result, error) {
	err := c.Update(context.TODO(), d)
	if err != nil {
		logger.Error(err, "Error updating dataset status")
	}
	return reconcile.Result{}, err
}

func readEventsForPVC(c *kubernetes.Clientset, pvc *v1.PersistentVolumeClaim, logger logr.Logger) (string, error) {
	lastMessage := ""

	listOpts := metav1.ListOptions{
		FieldSelector: fmt.Sprintf(
			"involvedObject.name=%s,"+
				"involvedObject.resourceVersion=%s,"+
				"involvedObject.kind=PersistentVolumeClaim,"+
				"reason=ProvisioningFailed",
			pvc.Name, pvc.ResourceVersion,
		),
	}
	eventList, err := c.CoreV1().Events(pvc.Namespace).List(listOpts)
	if err == nil {
		eventsLen := len(eventList.Items)
		if eventsLen > 0 {
			// Assuming that always the last event in the list will be
			// the latest?!
			lastMessage = eventList.Items[eventsLen-1].Message
		}
	} else {
		logger.Error(err, "Reading events failed")
	}
	return lastMessage, err
}

func datasetHasCachingAnnotationsSet(d *comv1alpha1.Dataset) (string, bool) {
	cachingPluginName := ""
	datasetHasCachingAnnotations := true
	for _, cachingLabel := range []string{"dlf-plugin-type", "dlf-plugin-name"} {
		_, exists := d.Annotations[cachingLabel]
		if !exists {
			datasetHasCachingAnnotations = false
			break
		}
	}

	if datasetHasCachingAnnotations {
		cachingPluginName = d.Annotations["dlf-plugin-name"]
	}

	return cachingPluginName, datasetHasCachingAnnotations
}

func getCachingPlugins(c client.Client) (*v1.PodList, error) {

	namespace := os.Getenv("OPERATOR_NAMESPACE")
	podList := &v1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(map[string]string{
			"dlf-plugin-type": "caching",
		}),
		client.HasLabels{"dlf-plugin-name"},
	}
	err := c.List(context.TODO(), podList, listOpts...)
	return podList, err
}

func formatToYaml(in interface{}) (string, error) {
	d, err := yaml.Marshal(&in)
	if err != nil {
		log.Error(err, "Error in marshaling")
		return "", err
	}
	return string(d), nil
}
