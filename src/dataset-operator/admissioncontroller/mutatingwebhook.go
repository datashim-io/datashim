// Package mutate deals with AdmissionReview requests and responses, it takes in the request body and returns a readily converted JSON []byte that can be
// returned from a http Handler w/o needing to further convert or modify it, it also makes testing Mutate() kind of easy w/o need for a fake http server, etc.
package admissioncontroller

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"

	datasetsv1alpha1 "github.com/datashim-io/datashim/src/dataset-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	prefixLabels   = "dataset."
	labelSeparator = "."
)

var (
	log = logf.Log.WithName("datashim-webhook")
)

//following the kubebuilder example for the pod mutator

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create,versions=v1,name=mpod.datashim.io, admissionReviewVersions=v1,sideEffects=NoneOnDryRun
type DatasetPodMutator struct {
	Client  client.Client
	Decoder admission.Decoder
}

func (m *DatasetPodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	// Mutate mutates
	log = log.WithValues("admission pod request", req)
	log.V(1).Info("webhook received", "request", req)

	if req.Operation != admissionv1.Create {
		return admission.Allowed(fmt.Sprintf("No Pod mutation required for operation %v.", req.Operation))
	}

	pod := &corev1.Pod{}

	err := m.Decoder.Decode(req, pod)

	if err != nil {
		log.Error(fmt.Errorf("could not decode pod %s", pod.Name), "could not decode pod")
		return admission.Errored(http.StatusBadRequest, err)
	}

	datasetInputs, err := DatasetInputFromPod(pod)

	if err != nil {
		log.Error(err, "could not retrieve datasets from pod spec", "pod", pod)
		return admission.Errored(http.StatusBadRequest, err)
	}

	log.V(1).Info("Pod spec contains datasets", "datasets", len(datasetInputs))

	if err := RetrieveDatasetsFromAPIServer(ctx, m.Client, pod, datasetInputs); err != nil {
		log.Error(err, "Error in dataset specification", "pod", pod, "error", err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	patchops, err := PatchPodWithDatasetLabels(pod, datasetInputs)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("useas for this dataset is not recognized"))
	}

	if len(patchops) == 0 {
		m := fmt.Sprintf("Pod %s/%s does not need to be mutated - skipping\n", pod.Namespace, pod.Name)
		log.V(0).Info("No datasets found", "Pod", pod)
		return admission.Allowed(m)
	}

	return admission.Patched("added volumes to Pod in response to dataset", patchops...)

}

// InjectDecoder injects the decoder.
func (m *DatasetPodMutator) InjectDecoder(d admission.Decoder) error {
	m.Decoder = d
	return nil
}

type DatasetInput struct {
	name        string
	index       int
	useasLabels []string
	resource    *datasetsv1alpha1.Dataset
}

func NewDatasetInput() *DatasetInput {
	return &DatasetInput{
		name:        "",
		useasLabels: []string{},
		resource:    &datasetsv1alpha1.Dataset{},
	}
}

func (d *DatasetInput) RequestedUseContains(useas string) bool {
	for _, u := range d.useasLabels {
		if u == useas {
			return true
		}
	}
	return false
}

func (d *DatasetInput) SetName(name string) *DatasetInput {
	d.name = name
	return d
}

func (d *DatasetInput) SetIndex(idx int) *DatasetInput {
	d.index = idx
	return d
}

func (d *DatasetInput) AddToRequestedUse(useas string) *DatasetInput {
	d.useasLabels = append(d.useasLabels, useas)
	return d
}

func (d *DatasetInput) String() string {
	return fmt.Sprintf("Dataset Input From Pod name: %s, useas: %v", d.name, d.useasLabels)
}

func DatasetInputFromPod(pod *corev1.Pod) (map[int]*DatasetInput, error) {
	// Format is {"id": {"index": <str>, "useas": mount/configmap}
	//log = log.WithName("dataset-label-processing")
	log.V(1).Info("Pod labels", "labels", pod.Labels)

	datasets := map[int]*DatasetInput{}
	unique_datasets := map[string]bool{}

	for k, v := range pod.Labels {
		log.V(1).Info("processing label", k, v)
		if strings.HasPrefix(k, prefixLabels) {
			log.V(1).Info("Dataset input label in pod")
			datasetNameArray := strings.Split(k, labelSeparator)
			if len(datasetNameArray) != 3 {
				err_out := fmt.Errorf("label %s is not in the right format", k)
				log.Error(err_out, "Format error in Dataset Labels", k, v)
				return nil, err_out
			}

			idx, err := strconv.Atoi(datasetNameArray[1])
			if err != nil {
				err_out := fmt.Errorf("could not convert dataset index %s to int", datasetNameArray[1])
				log.Error(err_out, "Format error in Dataset label", k, v)
				return nil, err_out
			}

			dataset, ok := datasets[idx]
			if !ok {
				log.V(1).Info("Did not find a dataset input for this index", "index", idx)
				dataset = NewDatasetInput()
			}
			switch datasetNameArray[2] {
			case "id":
				if dataset.name != "" {
					err_out := fmt.Errorf("repeat declaration of name %s for dataset %s", datasetNameArray[2], dataset.name)
					log.Error(err_out, "Format error in Dataset label", k, v)
					return nil, err_out
				} else if _, found := unique_datasets[v]; found {
					// We do not want the following scenario
					// dataset.0.id: "foo"
					// dataset.0.id: "mount"
					// dataset.1.id: "foo"
					// ...
					// In this case, we will have to process foo twice.
					// This is a format error for the Dataset
					err_out := fmt.Errorf("dataset name %s has been used in a previous label", v)
					return nil, err_out
				} else {
					dataset = dataset.SetName(v)
					unique_datasets[v] = true
				}
			case "useas":
				isPresent := false
				uArray := strings.Split(v, labelSeparator)
				log.V(1).Info("Dataset useas received", "useas", uArray)
				for _, use := range uArray {

					for _, u := range dataset.useasLabels {
						if u == strings.TrimSpace(use) {
							log.V(0).Info("Repeat declaration of useas in dataset label", k, v)
							isPresent = true
						}
					}
					if !isPresent {
						dataset = dataset.AddToRequestedUse(strings.TrimSpace(use))
					}
				}
			default:
				err_out := fmt.Errorf("dataset label is in the wrong format %s", k)
				log.Error(err_out, "Format error in Dataset label", k, v)
				return nil, err_out
			}

			datasets[idx] = dataset
		}
	}

	return datasets, nil
}

func RetrieveDatasetsFromAPIServer(ctx context.Context, client client.Client, pod *corev1.Pod, datasets map[int]*DatasetInput) error {
	for _, dataset := range datasets {
		log.V(1).Info("Checking dataset for validity", "Dataset", dataset)

		ds := &datasetsv1alpha1.Dataset{}
		nsName := types.NamespacedName{
			Namespace: pod.Namespace,
			Name:      dataset.name,
		}
		err := client.Get(context.TODO(), nsName, ds)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Error(err, "Dataset not found in the namespace", dataset.name, pod.Namespace)
				return fmt.Errorf("dataset %s not found in namespace %s", dataset.name, pod.Namespace)
			} else {
				log.Error(err, "Could not query API server", dataset.name, pod.Namespace)
				return fmt.Errorf("dataset %s could not be queried successfully", dataset.name)
			}
		} else {
			//TODO: Other things we want to check out
			// Does the Pod have the labels that is in the datasets allowed list
			// Does the backend for the dataset support the selected useas method
			log.V(1).Info("Found dataset", "Dataset.name", ds.Name, "Dataset.Spec", ds.Spec)
			// Store the dataset object
			dataset.resource = ds
		}
	}

	return nil
}

func PatchPodWithDatasetLabels(pod *corev1.Pod, datasets map[int]*DatasetInput) ([]jsonpatch.JsonPatchOperation, error) {
	//log = log.WithName("pod-patcher")
	patchops := []jsonpatch.JsonPatchOperation{}

	if len(datasets) == 0 {
		log.V(1).Info("no datasets were present in pod", "pod", pod.ObjectMeta.Name)
		return patchops, nil
	}

	// Record the names of already mounted PVCs. Cross-check them with those referenced in
	// a label. We only want to inject PVCs whose name is not in the mountedPVCs map.
	mountedPVCs := make(map[string]int)
	for _, v := range pod.Spec.Volumes {
		if v.PersistentVolumeClaim != nil {
			mountedPVCs[v.PersistentVolumeClaim.ClaimName] = 1
		}
	}

	datasets_tomount := map[int]*DatasetInput{}
	configs_toinject := map[int]string{}
	d, c := 0, 0
	s_idx := make([]int, 0, len(datasets))

	for idx := range datasets {
		s_idx = append(s_idx, idx)
	}
	sort.Ints(s_idx)
	for _, idx := range s_idx {

		//TODO: currently, the useas and dataset types are not cross-checked
		//e.g. useas configmap is not applicable to NFS shares.
		//There may be future dataset backends (e.g. SQL queries) that may
		//not be able to be mounted. This logic needs to be revisited
		ds := datasets[idx]
		log.V(1).Info("dataset label", "index", idx, "dataset", ds)
		for _, u := range ds.useasLabels {
			log.V(1).Info("Processing", "useas", u)
			switch u {
			case "mount":
				// The dataset is already mounted as a PVC no need to add it again
				log.V(1).Info("doing", "useas", u)

				if _, found := mountedPVCs[ds.name]; !found {
					log.V(1).Info("Adding to volumes to mount", "dataset", ds.name)
					datasets_tomount[d] = ds
					d += 1
				}
			case "configmap":
				//by default, we will mount a config map inside the containers.
				configs_toinject[c] = ds.name
				c += 1
			default:
				//this is an error
				log.V(1).Info("Error: The useas for this dataset is not recognized", "index", idx, "dataset", ds)
				return nil, fmt.Errorf("encountered an unknown useas")
			}
		}
	}

	log.V(1).Info("Num of patches", "mount", len(datasets_tomount), "configmaps", len(configs_toinject))

	if len(datasets_tomount) > 0 {
		log.V(1).Info("Patching volumes to Pod Spec", "datasets", datasets_tomount)
		patch_ds := patchPodSpecWithDatasetPVCs(pod, datasets_tomount, log)
		patchops = append(patchops, patch_ds...)
	}

	if len(configs_toinject) > 0 {
		log.V(1).Info("Adding config maps to Init Containers", "configmaps", configs_toinject)
		config_patch_init := patchContainersWithDatasetMaps(configs_toinject, pod.Spec.InitContainers, true)
		patchops = append(patchops, config_patch_init...)

		log.V(1).Info("Adding config maps to App Containers", "configmaps", configs_toinject)
		config_patch_main := patchContainersWithDatasetMaps(configs_toinject, pod.Spec.Containers, false)
		patchops = append(patchops, config_patch_main...)
	}

	return patchops, nil
}

func patchPodSpecWithDatasetPVCs(pod *corev1.Pod, datasets map[int]*DatasetInput, log logr.Logger) (patches []jsonpatch.JsonPatchOperation) {
	patches = []jsonpatch.JsonPatchOperation{}

	vol_id := len(pod.Spec.Volumes)
	init_containers := pod.Spec.InitContainers
	main_containers := pod.Spec.Containers
	log.V(1).Info("Num items for patching", "datasets", len(datasets), "init containers", len(init_containers), "main containers", len(main_containers))

	keys := make([]int, 0, len(datasets))
	for k := range datasets {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	log.V(1).Info("dataset indices", "index", keys)

	//This is done to satisfy testing as it is impossible to set expected outputs when the patches appear in random order
	for d := range keys {
		var pvc map[string]interface{}
		log.V(1).Info("Spec.Local", "spec.local", datasets[d].resource.Spec)
		readonly, ok := datasets[d].resource.Spec.Local["readonly"]
		ro := true
		if ok && readonly == "true" {
			log.V(1).Info("Read-only dataset", "name", datasets[d].name, "pod", pod.ObjectMeta.Name)
			pvc = map[string]interface{}{
				"claimName": datasets[d].name,
				"readOnly":  &ro,
			}
		} else {
			log.V(1).Info("Readwrite dataset", "name", datasets[d].name, "pod", pod.ObjectMeta.Name)
			pvc = map[string]interface{}{
				"claimName": datasets[d].name,
			}
		}
		patch := jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/spec/volumes/" + fmt.Sprint(vol_id),
			Value: map[string]interface{}{
				"name":                  datasets[d].name,
				"persistentVolumeClaim": pvc,
			},
		}
		patches = append(patches, patch)
		vol_id += 1
	}

	if len(init_containers) != 0 {
		log.V(1).Info("Patching init containers")
		volPatches := patchContainersWithDatasetVolumes(pod, datasets, keys, init_containers, true)
		patches = append(patches, volPatches...)
	}

	log.V(1).Info("Patching main containers")
	volPatches := patchContainersWithDatasetVolumes(pod, datasets, keys, main_containers, false)
	patches = append(patches, volPatches...)

	return patches
}

func patchContainersWithDatasetVolumes(pod *corev1.Pod, datasets map[int]*DatasetInput, order []int, containers []corev1.Container, init bool) (patches []jsonpatch.JsonPatchOperation) {

	patchOps := []jsonpatch.JsonPatchOperation{}

	container_typ := "containers"
	if init {
		container_typ = "initContainers"
		log.V(1).Info("Patching Init containers with datasets", "containers", containers, "datasets", datasets, "order", order)
	} else {
		log.V(1).Info("Patching Main containers with datasets", "containers", containers, "datasets", datasets, "order", order)
	}

	for container_idx, container := range containers {
		mounts := container.VolumeMounts
		mount_names := []string{}
		for _, mount := range mounts {
			mount_name := mount.Name
			mount_names = append(mount_names, mount_name)
		}
		mount_idx := len(mounts)

		for o := range order {
			exists, _ := in_array(datasets[o].name, mount_names)
			if !exists {
				log.V(1).Info("Dataset is not already mounted", "dataset", datasets[o], "pod", pod.Name)
				patch := jsonpatch.JsonPatchOperation{
					Operation: "add",
					Path:      "/spec/" + container_typ + "/" + fmt.Sprint(container_idx) + "/volumeMounts/" + fmt.Sprint(mount_idx),
					Value: map[string]interface{}{
						"name":      datasets[o].name,
						"mountPath": "/mnt/datasets/" + datasets[o].name,
					},
				}
				patchOps = append(patchOps, patch)
				mount_idx += 1
			} else {
				log.V(1).Info("Dataset is already mounted", "dataset", datasets[o], "pod", pod.Name)
			}
		}
	}
	return patchOps
}

func patchContainersWithDatasetMaps(datasets map[int]string, containers []corev1.Container, init bool) (patches []jsonpatch.JsonPatchOperation) {

	patchOps := []jsonpatch.JsonPatchOperation{}

	container_typ := "containers"

	if init {
		container_typ = "initContainers"
		log.V(1).Info("Patching Init containers with configmaps", "containers", containers, "configmaps", datasets)
	} else {
		log.V(1).Info("Patching Main containers with configmaps", "containers", containers, "configmaps", datasets)
	}

	var values []interface{}

	for container_idx, container := range containers {
		if container.EnvFrom != nil && len(container.EnvFrom) != 0 {
			// In this case, the envFrom path does exist in the PodSpec. So, we just append to
			// the existing array (Notice the path value)
			//Find existing configmap sources
			// We still assume that the configmaps are the same name as the dataset
			for _, d := range datasets {
				var found bool = false
				for _, env := range container.EnvFrom {
					if (env.ConfigMapRef != nil && env.ConfigMapRef.LocalObjectReference.Name == d) ||
						(env.SecretRef != nil && env.SecretRef.LocalObjectReference.Name == d) {
						found = true
						break
					}
				}
				if !found {
					cmPatchOp := jsonpatch.JsonPatchOperation{
						Operation: "add",
						Path:      "/spec/" + container_typ + "/" + fmt.Sprint(container_idx) + "/envFrom/-",
						Value: map[string]interface{}{
							"prefix": d + "_",
							"configMapRef": map[string]interface{}{
								"name": d,
							},
						},
					}
					patchOps = append(patchOps, cmPatchOp)
					secretPatchOp := jsonpatch.JsonPatchOperation{
						Operation: "add",
						Path:      "/spec/" + container_typ + "/" + fmt.Sprint(container_idx) + "/envFrom/-",
						Value: map[string]interface{}{
							"prefix": d + "_",
							"secretRef": map[string]interface{}{
								"name": d,
							},
						},
					}
					patchOps = append(patchOps, secretPatchOp)
				}
			}
		} else {
			for _, config_toinject := range datasets {
				//TODO: Check if the configmap reference exists in the API server
				// We also have to inject the companion secret. We are using the convention followed
				// in the controller where the names of the configmap and the secret are the same.
				configmap_ref := map[string]interface{}{
					"prefix": config_toinject + "_",
					"configMapRef": map[string]interface{}{
						"name": config_toinject,
					},
				}
				secret_ref := map[string]interface{}{
					"prefix": config_toinject + "_",
					"secretRef": map[string]interface{}{
						"name": config_toinject,
					},
				}

				values = append(values, configmap_ref)
				values = append(values, secret_ref)
			}

			// In this case, the envFrom path does not exist in the PodSpec. We are creating
			// (initialising) this path with an array of configMapRef (RFC 6902)
			patchOp := jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      "/spec/" + container_typ + "/" + fmt.Sprint(container_idx) + "/envFrom",
				Value:     values,
			}
			patchOps = append(patchOps, patchOp)
		}

	}

	return patchOps
}

func in_array(val interface{}, array interface{}) (exists bool, index int) {
	exists = false
	index = -1

	switch reflect.TypeOf(array).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(array)

		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(val, s.Index(i).Interface()) == true {
				index = i
				exists = true
				return
			}
		}
	}

	return
}
