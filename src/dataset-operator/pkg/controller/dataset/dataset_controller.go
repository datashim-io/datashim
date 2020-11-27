package dataset

import (
	"context"
	b64 "encoding/base64"
	comv1alpha1 "github.com/IBM/dataset-lifecycle-framework/src/dataset-operator/pkg/apis/com/v1alpha1"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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

	datasetInstance := &comv1alpha1.Dataset{}
	err := r.client.Get(context.TODO(), request.NamespacedName, datasetInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Dataset is deleted","name",request.NamespacedName)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	pluginPods, err := getCachingPlugins(r.client)
	if(err!=nil){return reconcile.Result{},err}

	// This means that we should create a 1-1 DatasetInteral
	if(len(pluginPods.Items)!=0){
		//TODO pick the first plugin for the time being
		datasetInstance.Annotations = pluginPods.Items[0].Labels
		err = r.client.Update(context.TODO(),datasetInstance)
		if(err!=nil){
			reqLogger.Error(err,"Error while updating dataset according to caching plugin")
			return reconcile.Result{},err
		}
		//In this case we are done, the caching plugin takes control of the dataset
		return reconcile.Result{}, nil
	}

	datasetInternalInstance := &comv1alpha1.DatasetInternal{}
	err = r.client.Get(context.TODO(), request.NamespacedName, datasetInternalInstance)
	if(err!=nil && !errors.IsNotFound(err)){
		//Unknown error occured, shouldn't happen
		return reconcile.Result{}, err
	} else if(err!=nil && errors.IsNotFound(err)){
			//1-1 Dataset and DatasetInternal because there is no caching plugin
			reqLogger.Info("1-1 Dataset and DatasetInternal because there is no caching plugin")

			newDatasetInternalInstance := &comv1alpha1.DatasetInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:                       datasetInstance.ObjectMeta.Name,
					Namespace:                  datasetInstance.ObjectMeta.Namespace,
				},
				Spec:       datasetInstance.Spec,
			}

			if(len(datasetInstance.Spec.Type) > 0 && datasetInstance.Spec.Type == "ARCHIVE") {
				podDownloadJob,bucket := getPodDataDownload(datasetInstance,os.Getenv("OPERATOR_NAMESPACE"))
				err = r.client.Create(context.TODO(),podDownloadJob)
				if(err!=nil){
					reqLogger.Error(err,"Error while creating pod download")
					return reconcile.Result{},err
				}
				minioConf := &v1.Secret{}
				err = r.client.Get(context.TODO(),types.NamespacedName{
					Namespace: os.Getenv("OPERATOR_NAMESPACE"),
					Name:      "minio-conf",
				},minioConf)
				if err != nil {
					reqLogger.Error(err,"Error while getting minio-conf secret")
					return reconcile.Result{},err
				}
				endpoint, _ := b64.StdEncoding.DecodeString(b64.StdEncoding.EncodeToString(minioConf.Data["ENDPOINT"]))
				accessKey, _ := b64.StdEncoding.DecodeString(b64.StdEncoding.EncodeToString(minioConf.Data["AWS_ACCESS_KEY_ID"]))
				secretAccessKey, _ := b64.StdEncoding.DecodeString(b64.StdEncoding.EncodeToString(minioConf.Data["AWS_SECRET_ACCESS_KEY"]))
				reqLogger.Info(string(endpoint))
				extract := "false"
				if(len(datasetInstance.Spec.Extract)>0) {
					extract = datasetInstance.Spec.Extract
				}
				newDatasetInternalInstance.Spec = comv1alpha1.DatasetSpec{
					Local: map[string]string{
						"type": "COS",
						"accessKeyID": string(accessKey),
						"secretAccessKey": string(secretAccessKey),
						"endpoint": string(endpoint),
						"readonly": "true",
						"bucket": bucket,
						"extract": extract,
						"region": "",
					},
				}
			}

			if err := controllerutil.SetControllerReference(datasetInstance, newDatasetInternalInstance, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			err = r.client.Create(context.TODO(),newDatasetInternalInstance)
			if err != nil {
				return reconcile.Result{}, err
			}
	}

	return reconcile.Result{}, nil
}

func getCachingPlugins(c client.Client) (*v1.PodList,error){

	namespace := os.Getenv("OPERATOR_NAMESPACE")
	podList := &v1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(map[string]string{
			"dlf-plugin-type":"caching",
		}),
		client.HasLabels{"dlf-plugin-name"},
	}
	err := c.List(context.TODO(),podList,listOpts...)
	return podList,err
}

func formatToYaml(in interface{}) (string,error) {
	d, err := yaml.Marshal(&in)
	if err != nil {
		log.Error(err,"Error in marshaling")
		return "", err
	}
	return string(d),nil
}
