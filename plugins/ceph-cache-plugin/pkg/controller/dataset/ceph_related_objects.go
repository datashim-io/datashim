package dataset

import (
	"context"
	comv1alpha1 "github.com/IBM/dataset-lifecycle-framework/src/dataset-operator/pkg/apis/com/v1alpha1"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createCustomConfigMapForRados(c client.Client,dataset *comv1alpha1.Dataset) error{
	accessKeyID := dataset.Spec.Local["accessKeyID"]
	secretAccessKey := dataset.Spec.Local["secretAccessKey"]
	endpoint := dataset.Spec.Local["endpoint"]
	bucket := dataset.Spec.Local["bucket"]

	configMapForRados := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-rgw-"+dataset.Name+"-custom",
			Namespace: os.Getenv("ROOK_NAMESPACE"),
			Labels: map[string]string{
				"dataset": dataset.Name,
				"dataset-namespace": dataset.Namespace,
				"dataset-uid": string(dataset.UID),
			},
		},
		Data: map[string]string{
			"config": "\n[global]\nrgw frontends = civetweb port=8000\nadmin socket = /tmp/radosgw.8000.asok\nremote s3 = "+endpoint+"\nremote bucket = "+bucket+"\nremote id = "+accessKeyID+"\nremote secret = "+secretAccessKey+"\n",
		},
	}
	err := c.Create(context.TODO(),	configMapForRados)
	return err;
}

func createCephObjectStore(c client.Client,dataset *comv1alpha1.Dataset) error{

	log.Info("Creating ceph object store")

	newRgw := &cephv1.CephObjectStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataset.ObjectMeta.Name,
			Namespace: os.Getenv("ROOK_NAMESPACE"),
			Labels: map[string]string{
				"dataset": dataset.Name,
				"dataset-namespace": dataset.Namespace,
				"dataset-uid": string(dataset.UID),
			},
		},
		Spec:       cephv1.ObjectStoreSpec{
			MetadataPool:          cephv1.PoolSpec{
				FailureDomain: "host",
				Replicated:      cephv1.ReplicatedSpec{
					Size:                   1,
				},
			},
			DataPool:              cephv1.PoolSpec{
				FailureDomain: "host",
				Replicated:      cephv1.ReplicatedSpec{
					Size:                   1,
				},
			},
			PreservePoolsOnDelete: false,
			Gateway:               cephv1.GatewaySpec{
				Instances: 1,
				Port: 80,
			},
		},
	}
	err := c.Create(context.TODO(),newRgw)
	return err;
}

func createCephObjectStoreUser(c client.Client, dataset *comv1alpha1.Dataset) error{

	log.Info("Creating ceph object store user")

	cephObjectStoreUser := &cephv1.CephObjectStoreUser{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataset.ObjectMeta.Name,
			Namespace: os.Getenv("ROOK_NAMESPACE"),
			Labels: map[string]string{
				"dataset": dataset.Name,
				"dataset-namespace": dataset.Namespace,
				"dataset-uid": string(dataset.UID),
			},
		},
		Spec:       cephv1.ObjectStoreUserSpec{
			Store:       dataset.ObjectMeta.Name,
			DisplayName: "Ceph User",
		},
	}

	err := c.Create(context.TODO(),cephObjectStoreUser)
	return err;
}

func isSameCephObject(labels map[string]string, datasetInstance *comv1alpha1.Dataset) (bool){
	sameObj := true
	if(labels==nil){
		sameObj = false
		log.Info("Doesn't have labels")
	} else {
		if(labels["dataset"]!=datasetInstance.ObjectMeta.Name){
			sameObj = false
			log.Info("Not the same dataset name")
			log.Info("Dataset name "+datasetInstance.ObjectMeta.Name)
		} else if(labels["dataset-namespace"]!=datasetInstance.ObjectMeta.Namespace){
			sameObj = false
			log.Info("Not the same dataset namespace")
			log.Info("Dataset namespace "+datasetInstance.ObjectMeta.Name)
		} else if(labels["dataset-uid"]!=string(datasetInstance.ObjectMeta.UID)) {
			sameObj = false
			log.Info("Not the same dataset uid")
			log.Info("Dataset uid "+string(datasetInstance.ObjectMeta.UID))
		}
	}
	return sameObj
}