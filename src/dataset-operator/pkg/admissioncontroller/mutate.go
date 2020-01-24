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

	if ar != nil {

		// get the Pod object and unmarshal it into its struct, if we cannot, we might as well stop here
		if err := json.Unmarshal(ar.Object.Raw, &pod); err != nil {
			return nil, fmt.Errorf("unable unmarshal pod json object %v", err)
		}

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

		p := []map[string]interface{}{}

		for k, v := range datasetInfo {
			log.Printf("key[%s] value[%s]\n", k, v)
			patch := map[string]interface{}{
				"op":   "add",
				"path": "/spec/volumes/" + fmt.Sprint(existing_volumes_id),
			}

			switch v["useas"] {
			case "mount":
				patch["value"] = map[string]interface{}{
					"name":                  v["id"],
					"persistentVolumeClaim": map[string]string{"claimName": v["id"]},
				}
			case "configmap":
				//by default, we will mount a config map inside the containers.
				fallthrough
			default:
				//We will mount the configmap as a volume as well
				patch["value"] = map[string]interface{}{
					"name": v["id"],
					"configMap": map[string]string{
						"name": v["id"],
					},
				}
			}

			datasets_tomount = append(datasets_tomount, v["id"])
			p = append(p, patch)
			existing_volumes_id += 1
		}

		containers := pod.Spec.Containers
		for container_idx, container := range containers {
			mounts := container.VolumeMounts
			mount_names := []string{}
			for _, mount := range mounts {
				mount_name := mount.Name
				mount_names = append(mount_names, mount_name)
			}
			mount_idx := len(mounts)

			for _, dataset_tomount := range datasets_tomount {
				exists, _ := in_array(dataset_tomount, mount_names)
				if exists == false {
					patch := map[string]interface{}{
						"op":   "add",
						"path": "/spec/containers/" + fmt.Sprint(container_idx) + "/volumeMounts/" + fmt.Sprint(mount_idx),
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

		resp.Patch, err = json.Marshal(p)

		// Success, of course ;)
		resp.Result = &metav1.Status{
			Status: "Success",
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
