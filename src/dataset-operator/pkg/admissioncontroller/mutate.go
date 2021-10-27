// Package mutate deals with AdmissionReview requests and responses, it takes in the request body and returns a readily converted JSON []byte that can be
// returned from a http Handler w/o needing to further convert or modify it, it also makes testing Mutate() kind of easy w/o need for a fake http server, etc.
package admissioncontroller

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	v1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	prefixLabels = "dataset."
)

// Mutate mutates
func Mutate(body []byte) ([]byte, error) {

	log.Printf("recv: %s\n", string(body))

	// unmarshal request into AdmissionReview struct
	admReview := v1beta1.AdmissionReview{}
	if err := json.Unmarshal(body, &admReview); err != nil {
		return nil, fmt.Errorf("unmarshaling request failed with %s", err)
	}

	var err error
	var pod *corev1.Pod

	responseBody := []byte{}
	ar := admReview.Request
	resp := v1beta1.AdmissionResponse{}
	admReview.Response = &resp

	if ar != nil {

		// get the Pod object and unmarshal it into its struct, if we cannot, we might as well stop here
		if err := json.Unmarshal(ar.Object.Raw, &pod); err != nil {
			return nil, fmt.Errorf("unable unmarshal pod json object %v", err)
		}

		// Record the names of already mounted PVCs. Cross-check them with those referenced in
		// a label. We only want to inject PVCs whose name is not in the mountedPVCs map.
		mountedPVCs := make(map[string]int)
		for _, v := range pod.Spec.Volumes {
			if v.PersistentVolumeClaim != nil {
				mountedPVCs[v.PersistentVolumeClaim.ClaimName] = 1
			}
		}

		datasetInfo := map[string]map[string]string{}

		for k, v := range pod.Labels {
			log.Printf("key[%s] value[%s]\n", k, v)
			if strings.HasPrefix(k, prefixLabels) {
				if _, found := mountedPVCs[v]; found {
					// The dataset is already mounted as a PVC no need to add it again
					continue
				}

				datasetNameArray := strings.Split(k, ".")
				datasetId := strings.Join([]string{datasetNameArray[0], datasetNameArray[1]}, ".")
				if _, ok := datasetInfo[datasetId]; ok == false {
					datasetInfo[datasetId] = map[string]string{datasetNameArray[2]: v}
				} else {
					datasetInfo[datasetId][datasetNameArray[2]] = v
				}
			}
		}

		if len(datasetInfo) == 0 {
			log.Printf("Pod %s/%s does not need to be mutated - skipping\n", pod.Namespace, pod.Name)
			resp.Allowed = true
			resp.UID = ar.UID
			responseBody, err = json.Marshal(admReview)
			if err != nil {
				return nil, err
			}
			return responseBody, nil
		}
		// set response options
		resp.Allowed = true
		resp.UID = ar.UID
		pT := v1beta1.PatchTypeJSONPatch
		resp.PatchType = &pT // it's annoying that this needs to be a pointer as you cannot give a pointer to a constant?

		// add some audit annotations, helpful to know why a object was modified, maybe (?)
		resp.AuditAnnotations = map[string]string{
			"mutateme": "yup it did it",
		}

		existing_volumes_id := len(pod.Spec.Volumes)
		datasets_tomount := []string{}
		configs_toinject := []string{}

		p := []map[string]interface{}{}

		for k, v := range datasetInfo {
			log.Printf("key[%s] value[%s]\n", k, v)

			//TODO: currently, the useas and dataset types are not cross-checked
			//e.g. useas configmap is not applicable to NFS shares.
			//There may be future dataset backends (e.g. SQL queries) that may
			//not be able to be mounted. This logic needs to be revisited
			switch v["useas"] {
			case "mount":
				patch := map[string]interface{}{
					"op":   "add",
					"path": "/spec/volumes/" + fmt.Sprint(existing_volumes_id),
				}
				patch["value"] = map[string]interface{}{
					"name":                  v["id"],
					"persistentVolumeClaim": map[string]string{"claimName": v["id"]},
				}
				datasets_tomount = append(datasets_tomount, v["id"])
				p = append(p, patch)
				existing_volumes_id += 1
			case "configmap":
				//by default, we will mount a config map inside the containers.
				configs_toinject = append(configs_toinject, v["id"])
			default:
				//this is an error
				log.Printf("Error: The useas for this dataset is not recognized")
			}

		}

		init_containers := pod.Spec.InitContainers
		main_containers := pod.Spec.Containers

		log.Printf("There are %d init containers and %d main containers", len(init_containers), len(main_containers))

		if len(datasets_tomount) > 0 {
			log.Print("Adding Volumes to Init Containers")
			patch_init := patchContainersWithDatasetVolumes(datasets_tomount, init_containers, true)
			p = append(p, patch_init...)

			log.Print("Adding Volumes to App Containers")
			patch_main := patchContainersWithDatasetVolumes(datasets_tomount, main_containers, false)
			p = append(p, patch_main...)
		}

		if len(configs_toinject) > 0 {
			log.Print("Adding config maps to Init Containers")
			config_patch_init := patchContainersWithDatasetMaps(configs_toinject, init_containers, true)
			p = append(p, config_patch_init...)

			log.Print("Adding config maps to App Containers")
			config_patch_main := patchContainersWithDatasetMaps(configs_toinject, main_containers, false)
			p = append(p, config_patch_main...)
		}

		log.Printf("Patch \n%v", p)
		resp.Patch, err = json.Marshal(p)

		// Success, of course ;)
		if err != nil {
			return nil, err
		} else {
			resp.Result = &metav1.Status{
				Status: "Success",
			}
		}

		admReview.Response = &resp
		// back into JSON so we can return the finished AdmissionReview w/ Response directly
		// w/o needing to convert things in the http handler
		responseBody, err = json.Marshal(admReview)
		if err != nil {
			return nil, err
		}
	}

	log.Printf("resp: %s\n", string(responseBody))
	return responseBody, nil
}

func patchContainersWithDatasetVolumes(datasets []string, containers []corev1.Container, init bool) (patch []map[string]interface{}) {

	p := []map[string]interface{}{}
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
				patch := map[string]interface{}{
					"op":   "add",
					"path": "/spec/" + container_typ + "/" + fmt.Sprint(container_idx) + "/volumeMounts/" + fmt.Sprint(mount_idx),
					"value": map[string]interface{}{
						"name":      dataset_tomount,
						"mountPath": "/mnt/datasets/" + dataset_tomount,
					},
				}
				p = append(p, patch)
				mount_idx += 1
			}
		}
	}

	return p

}

func patchContainersWithDatasetMaps(datasets []string, containers []corev1.Container, init bool) (patch []map[string]interface{}) {

	p := []map[string]interface{}{}

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
			patch := map[string]interface{}{
				"op":    "add",
				"path":  "/spec/" + container_typ + "/" + fmt.Sprint(container_idx) + "/envFrom",
				"value": values,
			}
			p = append(p, patch)
		} else {
			// In this case, the envFrom path does exist in the PodSpec. So, we just append to
			// the existing array (Notice the path value)
			for _, val := range values {
				patch := map[string]interface{}{
					"op":    "add",
					"path":  "/spec/" + container_typ + "/" + fmt.Sprint(container_idx) + "/envFrom/-",
					"value": val,
				}
				p = append(p, patch)
			}
		}

	}

	return p
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
