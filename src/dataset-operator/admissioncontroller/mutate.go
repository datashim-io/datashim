// Package mutate deals with AdmissionReview requests and responses, it takes in the request body and returns a readily converted JSON []byte that can be
// returned from a http Handler w/o needing to further convert or modify it, it also makes testing Mutate() kind of easy w/o need for a fake http server, etc.
package admissioncontroller

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"

	jsonpatch "gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	prefixLabels = "dataset."
)

//following the kubebuilder example for the pod mutator

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.datashim.io
type DatasetPodMutator struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (m *DatasetPodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	// Mutate mutates
	log.Printf("recv: %s\n", string(req.String()))

	var err error
	pod := &corev1.Pod{}

	err = m.decoder.Decode(req, pod)

	if err != nil {
		log.Fatalf("Could not decode pod ")
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Record the names of already mounted PVCs. Cross-check them with those referenced in
	// a label. We only want to inject PVCs whose name is not in the mountedPVCs map.

	// Format is {<dataset id:str>: 1} -- basically using the map as a set
	mountedPVCs := make(map[string]int)
	for _, v := range pod.Spec.Volumes {
		if v.PersistentVolumeClaim != nil {
			mountedPVCs[v.PersistentVolumeClaim.ClaimName] = 1
		}
	}

	// Format is {"dataset.<index>": {"id": <str>, "useas": mount/configmap}
	datasetInfo := map[string]map[string]string{}

	for k, v := range pod.Labels {
		log.Printf("key[%s] value[%s]\n", k, v)
		if strings.HasPrefix(k, prefixLabels) {
			datasetNameArray := strings.Split(k, ".")
			datasetId := strings.Join([]string{datasetNameArray[0], datasetNameArray[1]}, ".")
			if _, ok := datasetInfo[datasetId]; ok == false {
				datasetInfo[datasetId] = map[string]string{datasetNameArray[2]: v}
			} else {
				datasetInfo[datasetId][datasetNameArray[2]] = v
			}
		}
	}
	// Finally, don't inject those datasets which are already mounted as a PVC
	for datasetIndex, info := range datasetInfo {
		if info["useas"] == "mount" {
			// The dataset is already mounted as a PVC no need to add it again
			if _, found := mountedPVCs[info["id"]]; found {
				delete(datasetInfo, datasetIndex)
			}
		}
	}

	if len(datasetInfo) == 0 {
		m := fmt.Sprintf("Pod %s/%s does not need to be mutated - skipping\n", pod.Namespace, pod.Name)
		log.Printf(m)
		return admission.Allowed(m)
	}

	existing_volumes_id := len(pod.Spec.Volumes)
	datasets_tomount := []string{}
	configs_toinject := []string{}

	patchops := []jsonpatch.JsonPatchOperation{}

	for k, v := range datasetInfo {
		log.Printf("key[%s] value[%s]\n", k, v)

		//TODO: currently, the useas and dataset types are not cross-checked
		//e.g. useas configmap is not applicable to NFS shares.
		//There may be future dataset backends (e.g. SQL queries) that may
		//not be able to be mounted. This logic needs to be revisited
		switch v["useas"] {
		case "mount":
			patchop := jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      "/spec/volumes/" + fmt.Sprint(existing_volumes_id),
				Value: map[string]interface{}{
					"name":                  v["id"],
					"persistentVolumeClaim": map[string]string{"claimName": v["id"]},
				},
			}
			patchops = append(patchops, patchop)
			datasets_tomount = append(datasets_tomount, v["id"])
			existing_volumes_id += 1
		case "configmap":
			//by default, we will mount a config map inside the containers.
			configs_toinject = append(configs_toinject, v["id"])
		default:
			//this is an error
			log.Printf("Error: The useas for this dataset is not recognized")
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("The useas for this dataset is not recognized"))
		}

		init_containers := pod.Spec.InitContainers
		main_containers := pod.Spec.Containers

		log.Printf("There are %d init containers and %d main containers", len(init_containers), len(main_containers))

		if len(datasets_tomount) > 0 {
			log.Print("Adding Volumes to Init Containers")
			patch_init := patchContainersWithDatasetVolumes(datasets_tomount, init_containers, true)
			patchops = append(patchops, patch_init...)

			log.Print("Adding Volumes to App Containers")
			patch_main := patchContainersWithDatasetVolumes(datasets_tomount, main_containers, false)
			patchops = append(patchops, patch_main...)
		}

		if len(configs_toinject) > 0 {
			log.Print("Adding config maps to Init Containers")
			config_patch_init := patchContainersWithDatasetMaps(configs_toinject, init_containers, true)
			patchops = append(patchops, config_patch_init...)

			log.Print("Adding config maps to App Containers")
			config_patch_main := patchContainersWithDatasetMaps(configs_toinject, main_containers, false)
			patchops = append(patchops, config_patch_main...)
		}

	}

	return admission.Patched("added volumes to Pod in response to dataset", patchops...)

}

// InjectDecoder injects the decoder.
func (m *DatasetPodMutator) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d
	return nil
}

func patchContainersWithDatasetVolumes(datasets []string, containers []corev1.Container, init bool) (patches []jsonpatch.JsonPatchOperation) {

	patchOps := []jsonpatch.JsonPatchOperation{}

	container_typ := "containers"
	if init {
		container_typ = "initContainers"
	}

	for container_idx, container := range containers {
		mounts := container.VolumeMounts
		mount_names := []string{}
		for _, mount := range mounts {
			mount_name := mount.Name
			mount_names = append(mount_names, mount_name)
		}
		mount_idx := len(mounts)

		for _, dataset_tomount := range datasets {
			//TODO: Check if the dataset reference exists in the API server
			exists, _ := in_array(dataset_tomount, mount_names)
			if !exists {
				patch := jsonpatch.JsonPatchOperation{
					Operation: "add",
					Path:      "/spec/" + container_typ + "/" + fmt.Sprint(container_idx) + "/volumeMounts/" + fmt.Sprint(mount_idx),
					Value: map[string]interface{}{
						"name":      dataset_tomount,
						"mountPath": "/mnt/datasets/" + dataset_tomount,
					},
				}
				patchOps = append(patchOps, patch)
				mount_idx += 1
			}
		}
	}

	return patchOps

}

func patchContainersWithDatasetMaps(datasets []string, containers []corev1.Container, init bool) (patches []jsonpatch.JsonPatchOperation) {

	patchOps := []jsonpatch.JsonPatchOperation{}

	container_typ := "containers"
	if init {
		container_typ = "initContainers"
	}

	for container_idx, container := range containers {
		var values []interface{}
		for _, config_toinject := range datasets {
			//TODO: Check if the configmap reference exists in the API server

			configmap_ref := map[string]interface{}{
				"prefix": config_toinject + "_",
				"configMapRef": map[string]interface{}{
					"name": config_toinject,
				},
			}
			// We also have to inject the companion secret. We are using the convention followed
			// in the controller where the names of the configmap and the secret are the same.
			secret_ref := map[string]interface{}{
				"prefix": config_toinject + "_",
				"secretRef": map[string]interface{}{
					"name": config_toinject,
				},
			}

			values = append(values, configmap_ref)
			values = append(values, secret_ref)
		}

		if container.EnvFrom == nil || len(container.EnvFrom) == 0 {
			// In this case, the envFrom path does not exist in the PodSpec. We are creating
			// (initialising) this path with an array of configMapRef (RFC 6902)
			patchOp := jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      "/spec/" + container_typ + "/" + fmt.Sprint(container_idx) + "/envFrom",
				Value:     values,
			}
			patchOps = append(patchOps, patchOp)
		} else {
			// In this case, the envFrom path does exist in the PodSpec. So, we just append to
			// the existing array (Notice the path value)
			for _, val := range values {
				patchOp := jsonpatch.JsonPatchOperation{
					Operation: "add",
					Path:      "/spec/" + container_typ + "/" + fmt.Sprint(container_idx) + "/envFrom/-",
					Value:     val,
				}
				patchOps = append(patchOps, patchOp)
			}
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
