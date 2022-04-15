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
	"context"
	b64 "encoding/base64"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	datasets "github.com/datashim-io/datashim/api/v1alpha1"
	"github.com/go-logr/logr"
)

const (
	NameInvalidCharacters = "Name must consist of lower case alphanumeric characters or '-', and must start and " +
		"end with an alphanumeric character (e.g. 'example-dataset',  or '123-dataset')"
)

// DatasetReconciler reconciles a Dataset object
type DatasetReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Clientset *kubernetes.Clientset
}

//+kubebuilder:rbac:groups=datasets.datashim.io,resources=datasets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=datasets.datashim.io,resources=datasets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=datasets.datashim.io,resources=datasets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Dataset object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *DatasetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Dataset")

	datasetInstance := &datasets.Dataset{}
	err := r.Get(context.TODO(), req.NamespacedName, datasetInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Dataset is deleted", "name", req.NamespacedName)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Make sure that the Status of the Dataset is initialised
	if initializeDatasetStatus(datasetInstance) {

		// dataset.Name must consist of lower case alphanumeric characters or '-', and must start and end with an
		// alphanumeric character
		reg := regexp.MustCompile(`[a-z0-9]([-a-z0-9]*[a-z0-9])?`)
		if len(reg.ReplaceAllString(datasetInstance.Name, "")) > 0 {
			datasetInstance.Status.Provision.Status = datasets.StatusFail
			datasetInstance.Status.Provision.Info = NameInvalidCharacters
			datasetInstance.Status.Caching.Status = datasets.StatusFail
		}

		return updateDatasetStatusAndReturn(r.Client, datasetInstance, reqLogger)
	}

	// Caching setup
	if datasetInstance.Status.Caching.Status == datasets.StatusInitial {

		cacheDisableLabel, err := strconv.ParseBool(datasetInstance.Labels["cache.disable"])

		if err != nil {
			cacheDisableLabel = false
		}

		if cacheDisableLabel {
			// mark caching status as disabled and requeue
			datasetInstance.Status.Caching.Status = datasets.StatusDisabled
			datasetInstance.Status.Caching.Info = "User explicitly disabled caching"
			return updateDatasetStatusAndReturn(r.Client, datasetInstance, reqLogger)
		}

		// get the installed DLF caching plugins
		pluginPods, err := getCachingPlugins(r.Client)
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
					return updateDatasetAndReturn(r.Client, datasetInstance, reqLogger)
				}

				//User has specified the cache.plugin label
				for _, pluginItems := range pluginPods.Items {
					if pluginItems.Labels["dlf-plugin-name"] == cachePluginLabel {
						// Found the user specified plugin. Insert the proper annotations
						// and requeue
						datasetInstance.Annotations = pluginItems.Labels
						return updateDatasetAndReturn(r.Client, datasetInstance, reqLogger)
					}
				}

				// User specified plugin not found. Disable caching and requeue
				datasetInstance.Status.Caching.Status = datasets.StatusDisabled
				datasetInstance.Status.Caching.Info = fmt.Sprintf(
					"User specified plugin '%s' was not found. Falling back to disabled caching",
					cachePluginLabel)
				return updateDatasetStatusAndReturn(r.Client, datasetInstance, reqLogger)

			}

			// Dataset has the caching annotation embedded so now update accordingly
			// the status and reconcile
			// TODO the caching plugin is responsible for updating the caching status to OK
			datasetInstance.Status.Caching.Status = datasets.StatusPending
			datasetInstance.Status.Caching.Info = fmt.Sprintf(
				"Caching is assigned to %s plugin",
				cachingPluginName)
			return updateDatasetStatusAndReturn(r.Client, datasetInstance, reqLogger)
		}

		datasetInstance.Status.Caching.Status = datasets.StatusDisabled
		datasetInstance.Status.Caching.Info = "No DLF caching plugins are installed"
		return updateDatasetStatusAndReturn(r.Client, datasetInstance, reqLogger)
	}

	// We need to create DatasetInternal when caching is disabled for this Dataset
	// otherwise DatasetInternal is handled/created by the caching-plugin
	if datasetInstance.Status.Caching.Status == datasets.StatusDisabled {
		datasetInternalInstance := &datasets.DatasetInternal{}
		err = r.Get(context.TODO(), req.NamespacedName, datasetInternalInstance)
		if err != nil && !errors.IsNotFound(err) {
			//Unknown error occured, shouldn't happen
			return reconcile.Result{}, err
		} else if err != nil && errors.IsNotFound(err) {
			//1-1 Dataset and DatasetInternal because there is no caching plugin
			reqLogger.Info("1-1 Dataset and DatasetInternal because there is no caching plugin")

			newDatasetInternalInstance := datasets.DatasetInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      datasetInstance.ObjectMeta.Name,
					Namespace: datasetInstance.ObjectMeta.Namespace,
					Labels:    datasetInstance.ObjectMeta.Labels,
				},
				Spec: datasetInstance.Spec,
			}

			if len(datasetInstance.Spec.Type) > 0 && datasetInstance.Spec.Type == "ARCHIVE" {
				podDownloadJob, bucket := getPodDataDownload(datasetInstance, os.Getenv("OPERATOR_NAMESPACE"))
				err = r.Create(context.TODO(), podDownloadJob)
				if err != nil {
					reqLogger.Error(err, "Error while creating pod download")
					return reconcile.Result{}, err
				}
				minioConf := &v1.Secret{}
				err = r.Get(context.TODO(), types.NamespacedName{
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
				newDatasetInternalInstance.Spec = datasets.DatasetSpec{
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

			if err := controllerutil.SetControllerReference(datasetInstance, &newDatasetInternalInstance, r.Scheme); err != nil {
				return reconcile.Result{}, err
			}
			err = r.Create(context.TODO(), &newDatasetInternalInstance)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Provision Status
	// TODO for now for every Dataset kind we create a PVC, so the following is OK
	// However, we should make this check for PVC status only if the Dataset kind
	// is coupled with a PVC
	if datasetInstance.Status.Provision.Status != datasets.StatusFail &&
		datasetInstance.Status.Provision.Status != datasets.StatusOK {

		foundPVC := &v1.PersistentVolumeClaim{}
		err = r.Get(context.TODO(), req.NamespacedName, foundPVC)
		if err != nil {
			if errors.IsNotFound(err) {
				// If PVC is not found yet, reconcile with some delay
				if datasetInstance.Status.Provision.Status != datasets.StatusPending {
					datasetInstance.Status.Provision.Status = datasets.StatusPending
					return updateDatasetStatusAndReturn(r.Client, datasetInstance, reqLogger)
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
			if datasetInstance.Status.Provision.Status != datasets.StatusPending {
				datasetInstance.Status.Provision.Status = datasets.StatusPending
				statusUpdated = true
			}
			lastEvent, err := readEventsForPVC(ctx, r.Clientset, foundPVC, reqLogger)
			if err == nil && datasetInstance.Status.Provision.Info != lastEvent {
				datasetInstance.Status.Provision.Info = lastEvent
				statusUpdated = true
			}

			if statusUpdated {
				return updateDatasetStatusAndReturn(r.Client, datasetInstance, reqLogger)
			}
			// In the case the Dataset Status is not modified requeue with some delay
			// to re-check the status
			return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil

		case v1.ClaimBound:
			// if our PVC is bound make sure that the Provision Status of Dataset is
			// shown as OK and requeue
			if datasetInstance.Status.Provision.Status != datasets.StatusOK {
				datasetInstance.Status.Provision.Status = datasets.StatusOK
				datasetInstance.Status.Provision.Info = ""
				return updateDatasetStatusAndReturn(r.Client, datasetInstance, reqLogger)
			}
		}
	}

	datasetInternalInstance := &datasets.DatasetInternal{}
	err = r.Get(context.TODO(), req.NamespacedName, datasetInternalInstance)
	if err == nil {
		reqLogger.Info("Dataset Internal Instance exists, check if different EDITABLE labels and update")
		errForDatasetInternalUpdate := checkIfEditableLabelsChangedAndUpdate(r.Client, reqLogger, datasetInstance, datasetInternalInstance)
		if errForDatasetInternalUpdate != nil {
			return reconcile.Result{}, errForDatasetInternalUpdate
		}
	}
	// everything OK don't requeue
	// TODO maybe requeue with some delay if we
	// need to monitor statuses

	return ctrl.Result{}, nil
}

func checkIfEditableLabelsChangedAndUpdate(c client.Client, reqLogger logr.Logger, dataset *datasets.Dataset, datasetInternal *datasets.DatasetInternal) error {
	editableLabels := []string{"remove-on-delete"}
	datasetLabels := dataset.ObjectMeta.Labels
	datasetInternalLabels := datasetInternal.ObjectMeta.Labels
	if datasetInternalLabels == nil {
		datasetInternalLabels = map[string]string{}
	}
	shouldUpdateOrDelete := false
	for _, label := range editableLabels {
		if _, labelExistsInDataset := datasetLabels[label]; labelExistsInDataset {
			if _, labelExistsInDatasetInternal := datasetInternalLabels[label]; labelExistsInDatasetInternal {
				reqLogger.Info("Interested label " + label + " changed")
				shouldUpdateOrDelete = true
				datasetInternalLabels[label] = datasetLabels[label]
			} else {
				reqLogger.Info("Interested label " + label + " added")
				shouldUpdateOrDelete = true
				datasetInternalLabels[label] = datasetLabels[label]
			}
		}
		if _, labelExistsInDatasetInternal := datasetInternalLabels[label]; labelExistsInDatasetInternal {
			if _, labelExistsInDataset := datasetLabels[label]; !labelExistsInDataset {
				reqLogger.Info("Interested label " + label + " deleted")
				shouldUpdateOrDelete = true
				delete(datasetInternalLabels, label)
			}
		}
	}
	if shouldUpdateOrDelete {
		datasetInternal.ObjectMeta.Labels = datasetInternalLabels
		err := c.Update(context.TODO(), datasetInternal)
		return err
	}
	return nil
}

func initializeDatasetStatus(d *datasets.Dataset) bool {
	modifiedDatasetStatus := false
	if d.Status.Provision.Status == datasets.StatusEmpty {
		d.Status.Provision.Status = datasets.StatusInitial
		modifiedDatasetStatus = true
	}
	if d.Status.Caching.Status == datasets.StatusEmpty {
		d.Status.Caching.Status = datasets.StatusInitial
		modifiedDatasetStatus = true
	}
	return modifiedDatasetStatus
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatasetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&datasets.Dataset{}).
		Complete(r)
}

func updateDatasetStatusAndReturn(c client.Client, d *datasets.Dataset, logger logr.Logger) (reconcile.Result, error) {
	err := c.Status().Update(context.TODO(), d)
	if err != nil {
		logger.Error(err, "Error updating dataset status")
	}
	return reconcile.Result{}, err
}

func updateDatasetAndReturn(c client.Client, d *datasets.Dataset, logger logr.Logger) (reconcile.Result, error) {
	err := c.Update(context.TODO(), d)
	if err != nil {
		logger.Error(err, "Error updating dataset status")
	}
	return reconcile.Result{}, err
}

func readEventsForPVC(ctx context.Context, c *kubernetes.Clientset, pvc *v1.PersistentVolumeClaim, logger logr.Logger) (string, error) {
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
	eventList, err := c.CoreV1().Events(pvc.Namespace).List(ctx, listOpts)
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

func datasetHasCachingAnnotationsSet(d *datasets.Dataset) (string, bool) {
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

func formatToYaml(in interface{}, logger logr.Logger) (string, error) {
	d, err := yaml.Marshal(&in)
	if err != nil {
		logger.Error(err, "Error in marshaling")
		return "", err
	}
	return string(d), nil
}
