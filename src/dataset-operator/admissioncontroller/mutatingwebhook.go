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

	jsonpatch "gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	prefixLabels   = "dataset."
	labelSeparator = "."
)

var (
	log = ctrl.Log.WithName("datashim-webhook")
)

//following the kubebuilder example for the pod mutator

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create,versions=v1,name=mpod.datashim.io, admissionReviewVersions=v1,sideEffects=NoneOnDryRun
type DatasetPodMutator struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (m *DatasetPodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	// Mutate mutates

	log = logf.FromContext(ctx)
	log.V(1).Info("webhook received", "request", req)

	var err error
	pod := &corev1.Pod{}

	err = m.decoder.Decode(req, pod)

	if err != nil {
		log.Error(fmt.Errorf("could not decode pod %s", pod.Name), "could not decode pod")
		return admission.Errored(http.StatusBadRequest, err)
	}

	patchops, err := patchPodWithDatasetLabels(pod)

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
func (m *DatasetPodMutator) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d
	return nil
}

type DatasetInput struct {
	name  string
	useas []string
}

func NewDatasetInput() *DatasetInput {
	return &DatasetInput{
		name:  "",
		useas: []string{},
	}
}

func (d *DatasetInput) UseasContains(useas string) bool {
	for _, u := range d.useas {
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

func (d *DatasetInput) AddToUseas(useas string) *DatasetInput {
	d.useas = append(d.useas, useas)
	return d
}

func (d *DatasetInput) String() string {
	return fmt.Sprintf("Dataset Input From Pod name: %s, useas: %v", d.name, d.useas)
}

func DatasetInputFromPod(pod *corev1.Pod) (map[int]*DatasetInput, error) {
	// Format is {"id": {"index": <str>, "useas": mount/configmap}
	log.V(1).Info("Pod labels", "labels", pod.Labels)

	datasets := map[int]*DatasetInput{}

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
				} else {
					dataset = dataset.SetName(v)
				}
			case "useas":
				isPresent := false
				uArray := strings.Split(v, labelSeparator)
				log.V(1).Info("Dataset useas received", "useas", uArray)
				for _, use := range uArray {

					for _, u := range dataset.useas {
						if u == strings.TrimSpace(use) {
							log.V(0).Info("Repeat declaration of useas in dataset label", k, v)
							isPresent = true
						}
					}
					if !isPresent {
						dataset = dataset.AddToUseas(strings.TrimSpace(use))
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

	log.V(1).Info("Pod spec contains datasets", "datasets", len(datasets))

	return datasets, nil
}

func patchPodWithDatasetLabels(pod *corev1.Pod) ([]jsonpatch.JsonPatchOperation, error) {
	patchops := []jsonpatch.JsonPatchOperation{}

	datasetInfo, err := DatasetInputFromPod(pod)

	if err != nil {
		log.Error(err, "Error in parsing dataset labels", "Pod", pod)
		return nil, err
	}

	if len(datasetInfo) == 0 {
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

	datasets_tomount := map[int]string{}
	configs_toinject := map[int]string{}
	d, c := 0, 0
	s_idx := make([]int, 0, len(datasetInfo))

	for idx := range datasetInfo {
		s_idx = append(s_idx, idx)
	}
	sort.Ints(s_idx)
	for _, idx := range s_idx {

		//TODO: currently, the useas and dataset types are not cross-checked
		//e.g. useas configmap is not applicable to NFS shares.
		//There may be future dataset backends (e.g. SQL queries) that may
		//not be able to be mounted. This logic needs to be revisited
		ds := datasetInfo[idx]
		log.V(1).Info("dataset label", "index", idx, "dataset", ds)
		for _, u := range ds.useas {
			log.V(1).Info("Processing", "useas", u)
			switch u {
			case "mount":
				// The dataset is already mounted as a PVC no need to add it again
				log.V(1).Info("doing", "useas", u)

				if _, found := mountedPVCs[ds.name]; !found {
					log.V(1).Info("Adding to volumes to mount", "dataset", ds.name)
					datasets_tomount[d] = ds.name
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
		patch_ds := patchPodSpecWithDatasetPVCs(pod, datasets_tomount)
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

func patchPodSpecWithDatasetPVCs(pod *corev1.Pod, datasets map[int]string) (patches []jsonpatch.JsonPatchOperation) {
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
		patch := jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/spec/volumes/" + fmt.Sprint(vol_id),
			Value: map[string]interface{}{
				"name":                  datasets[d],
				"persistentVolumeClaim": map[string]string{"claimName": datasets[d]},
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

func patchContainersWithDatasetVolumes(pod *corev1.Pod, datasets map[int]string, order []int, containers []corev1.Container, init bool) (patches []jsonpatch.JsonPatchOperation) {

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
			//TODO: Check if the dataset reference exists in the API server
			exists, _ := in_array(datasets[o], mount_names)
			if !exists {
				patch := jsonpatch.JsonPatchOperation{
					Operation: "add",
					Path:      "/spec/" + container_typ + "/" + fmt.Sprint(container_idx) + "/volumeMounts/" + fmt.Sprint(mount_idx),
					Value: map[string]interface{}{
						"name":      datasets[o],
						"mountPath": "/mnt/datasets/" + datasets[o],
					},
				}
				patchOps = append(patchOps, patch)
				mount_idx += 1
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
