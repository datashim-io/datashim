package testing

import (
	"fmt"
	"strconv"

	datasetsv1alpha1 "github.com/datashim-io/datashim/src/dataset-operator/api/v1alpha1"
	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// basic idea for templating K8s objects to be used in tests comes
// from https://github.com/kubernetes-sigs/jobset/blob/main/pkg/util/testing/wrappers.go
type PVCWrapper struct {
	corev1.PersistentVolumeClaim
}

func MakePVC(name string, namespace string) *PVCWrapper {
	return &PVCWrapper{
		corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		},
	}
}

func (p *PVCWrapper) Obj() corev1.PersistentVolumeClaim {
	return p.PersistentVolumeClaim
}

type SecretWrapper struct {
	corev1.Secret
}

func MakeSecret(name string, namespace string) *SecretWrapper {
	return &SecretWrapper{
		corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		},
	}
}

func (s *SecretWrapper) Obj() corev1.Secret {
	return s.Secret
}

type DatasetWrapper struct {
	datasetsv1alpha1.Dataset
}

func MakeDataset(name string, namespace string) *DatasetWrapper {
	return &DatasetWrapper{
		Dataset: datasetsv1alpha1.Dataset{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Dataset",
				APIVersion: "com.ie.ibm.hpsys/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: datasetsv1alpha1.DatasetSpec{},
		},
	}
}

func (d *DatasetWrapper) ToS3Dataset(bucket, endpoint, secret string, readonly bool) *DatasetWrapper {
	d.Spec = datasetsv1alpha1.DatasetSpec{
		Local: map[string]string{
			"type":        "COS",
			"bucket":      bucket,
			"endpoint":    endpoint,
			"secret-name": secret,
			"readonly":    strconv.FormatBool(readonly),
		},
	}
	return d
}

func (d *DatasetWrapper) Obj() datasetsv1alpha1.Dataset {
	return d.Dataset
}

type DatasetInternalWrapper struct {
	datasetsv1alpha1.DatasetInternal
}

func MakeDatasetInternal(name, namespace string) *DatasetInternalWrapper {
	return &DatasetInternalWrapper{
		DatasetInternal: datasetsv1alpha1.DatasetInternal{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DatasetInternal",
				APIVersion: "com.ie.ibm.hpsys/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		},
	}
}

func InternalFromDataset(d datasetsv1alpha1.Dataset) *DatasetInternalWrapper {
	return &DatasetInternalWrapper{
		DatasetInternal: datasetsv1alpha1.DatasetInternal{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DatasetInternal",
				APIVersion: "com.ie.ibm.hpsys/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      d.Name,
				Namespace: d.Namespace,
			},
			Spec: *d.Spec.DeepCopy(),
		},
	}
}

func (di *DatasetInternalWrapper) ToS3DatasetInternal(bucket, endpoint, secret string, readonly bool) *DatasetInternalWrapper {
	di.Spec = datasetsv1alpha1.DatasetSpec{
		Local: map[string]string{
			"type":        "COS",
			"bucket":      bucket,
			"endpoint":    endpoint,
			"secret-name": secret,
			"readonly":    strconv.FormatBool(readonly),
		},
	}
	return di
}

func (di *DatasetInternalWrapper) Obj() datasetsv1alpha1.DatasetInternal {
	return di.DatasetInternal
}

type ContainerWrapper struct {
	corev1.Container
}

func MakeContainer(name string) *ContainerWrapper {
	return &ContainerWrapper{
		corev1.Container{
			Name:         name,
			VolumeMounts: []corev1.VolumeMount{},
		},
	}
}

func (c *ContainerWrapper) AddVolumeMount(mountPath, name string) *ContainerWrapper {
	c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
		Name:      name,
		ReadOnly:  false,
		MountPath: mountPath,
	})

	return c
}

func (c *ContainerWrapper) Obj() corev1.Container {
	return c.Container
}

type PodWrapper struct {
	corev1.Pod
}

func MakePod(name, ns string) *PodWrapper {
	return &PodWrapper{
		corev1.Pod{
			TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: map[string]string{}},
			Spec: corev1.PodSpec{
				Volumes:        []corev1.Volume{},
				InitContainers: []corev1.Container{},
				Containers:     []corev1.Container{},
			},
			Status: corev1.PodStatus{},
		},
	}
}

func (p *PodWrapper) PodLabels(labels map[string]string) *PodWrapper {
	p.Labels = labels
	return p
}

func (p *PodWrapper) AddLabelToPodMetadata(key, value string) *PodWrapper {
	_, ok := p.ObjectMeta.Labels[key]
	if !ok {
		p.ObjectMeta.Labels[key] = value
	}
	return p
}

func (p *PodWrapper) AddContainerToPod(container corev1.Container) *PodWrapper {
	p.Spec.Containers = append(p.Spec.Containers, container)
	return p
}

func (p *PodWrapper) AddVolumeToPod(volume string) *PodWrapper {
	existingVolumes := p.Spec.Volumes
	p.Spec.Volumes = append(existingVolumes, corev1.Volume{
		Name: volume,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: volume,
			},
		},
	})
	return p
}

func (p *PodWrapper) Obj() corev1.Pod {
	return p.Pod
}

type JSONPatchOperationWrapper struct {
	jsonpatch.JsonPatchOperation
}

func MakeJSONPatchOperation() *JSONPatchOperationWrapper {
	return &JSONPatchOperationWrapper{
		jsonpatch.JsonPatchOperation{
			Operation: "",
			Path:      "",
			Value:     nil,
		},
	}
}

func (j *JSONPatchOperationWrapper) SetOperation(operation string) *JSONPatchOperationWrapper {
	j.Operation = operation
	return j
}

func (j *JSONPatchOperationWrapper) SetVolumeasPath(id int) *JSONPatchOperationWrapper {
	j.Path = "/spec/volumes/" + fmt.Sprint(id)
	return j
}

func (j *JSONPatchOperationWrapper) SetVolumeMountasPath(cont_typ string, cont_idx, mount_idx int) *JSONPatchOperationWrapper {
	j.Path = "/spec/" + cont_typ + "/" + fmt.Sprint(cont_idx) + "/volumeMounts/" + fmt.Sprint(mount_idx)
	return j
}

func (j *JSONPatchOperationWrapper) SetPVCasValue(pvc string) *JSONPatchOperationWrapper {
	j.Value = map[string]interface{}{
		"name":                  pvc,
		"persistentVolumeClaim": map[string]string{"claimName": pvc},
	}
	return j
}

func (j *JSONPatchOperationWrapper) SetVolumeMountasValue(dataset string) *JSONPatchOperationWrapper {
	j.Value = map[string]interface{}{
		"name":      dataset,
		"mountPath": "/mnt/datasets/" + dataset,
	}
	return j
}

func (j *JSONPatchOperationWrapper) SetNewConfigMapRefasPath(cont_typ string, cont_idx int) *JSONPatchOperationWrapper {
	j.Path = "/spec/" + cont_typ + "/" + fmt.Sprint(cont_idx) + "/envFrom"
	return j
}

func (j *JSONPatchOperationWrapper) AddToConfigMapRefasPath(cont_typ string, cont_idx int) *JSONPatchOperationWrapper {
	j.Path = "/spec/" + cont_typ + "/" + fmt.Sprint(cont_idx) + "/envFrom-"
	return j
}

func (j *JSONPatchOperationWrapper) AddConfigMapRefsToValue(configmap_names []string) *JSONPatchOperationWrapper {
	var ref []interface{}
	for _, c := range configmap_names {
		ref = append(ref, map[string]interface{}{
			"prefix": c + "_",
			"configMapRef": map[string]interface{}{
				"name": c,
			},
		})
	}
	switch x := j.Value.(type) {
	case nil:
		j.Value = ref
	case []interface{}:
		x = append(x, ref...)
		j.Value = x
	}
	return j
}

func (j *JSONPatchOperationWrapper) AddSecretRefsToValue(secret_names []string) *JSONPatchOperationWrapper {
	var ref []interface{}
	for _, s := range secret_names {
		ref = append(ref, map[string]interface{}{
			"prefix": s + "_",
			"secretRef": map[string]interface{}{
				"name": s,
			},
		})
	}
	switch x := j.Value.(type) {
	case nil:
		j.Value = ref
	case []interface{}:
		x = append(x, ref...)
		j.Value = x
	}
	return j
}

func (j *JSONPatchOperationWrapper) Obj() jsonpatch.JsonPatchOperation {
	return j.JsonPatchOperation
}
