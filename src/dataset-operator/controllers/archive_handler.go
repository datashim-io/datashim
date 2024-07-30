package controllers

import (
	"path"
	"strconv"

	datasets "github.com/datashim-io/datashim/src/dataset-operator/api/v1alpha1"
	"github.com/google/uuid"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getPodDataDownload(dataset *datasets.Dataset, operatorNamespace string) (*batchv1.Job, string) {
	uuid_forpod, _ := uuid.NewUUID()
	podId := uuid_forpod.String()
	fileUrl := dataset.Spec.Url
	fileName := path.Base(fileUrl)
	seconds := int32(1)
	extract, _ := strconv.ParseBool(dataset.Spec.Extract)
	command := []string{
		"/bin/sh", "-c",
		"mkdir -p /data/" + podId + " && " +
			"wget " + fileUrl + " -P" + " /tmp && " +
			"tar -xf /tmp/" + fileName + " -C /data/" + podId + " && " +
			"rm -rf /tmp/" + fileName + " && " +
			"aws s3api create-bucket --bucket " + podId + " --endpoint-url $ENDPOINT && " +
			"aws s3 cp /data/" + podId + " s3://" + podId + " --recursive --endpoint-url $ENDPOINT && " +
			"rm -rf /data",
	}
	if !extract {
		command = []string{
			"/bin/sh", "-c",
			"wget " + fileUrl + " -P" + " /tmp && " +
				"aws s3api create-bucket --bucket " + podId + " --endpoint-url $ENDPOINT && " +
				"aws s3 cp /tmp/" + fileName + " s3://" + podId + " --endpoint-url $ENDPOINT && " +
				"rm -rf  /tmp/" + fileName,
		}
	}
	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{{
			Image:           "quay.io/datashim-io/cos-uploader:latest",
			ImagePullPolicy: corev1.PullAlways,
			Name:            "cos-uploader",
			Command:         command,
			EnvFrom: []corev1.EnvFromSource{
				{
					SecretRef: &corev1.SecretEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "minio-conf",
						},
					},
				},
			},
		}},
		RestartPolicy: corev1.RestartPolicyNever,
	}
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cos-upload-" + podId[:4],
			Namespace: operatorNamespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &seconds,
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
			TTLSecondsAfterFinished: &seconds,
		},
	}
	return job, podId
}
