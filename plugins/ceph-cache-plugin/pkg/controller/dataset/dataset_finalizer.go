package dataset

import (
	"context"
	comv1alpha1 "github.com/IBM/dataset-lifecycle-framework/src/dataset-operator/pkg/apis/com/v1alpha1"
	"github.com/go-logr/logr"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileDataset) finalizeDataset(reqLogger logr.Logger, m *comv1alpha1.Dataset) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.

	cephObjectStoreUser := &cephv1.CephObjectStoreUser{}
	err := getExactlyOneObject(r.client,cephObjectStoreUser,m.ObjectMeta.Name,os.Getenv("ROOK_NAMESPACE"))
	if(errors.IsNotFound(err)){
		reqLogger.Info("cephObjectStoreUser not created yet, we don't have to delete anything")
	} else if(err!=nil) {
		reqLogger.Info("Generic error for getting ceph object storeUser, shouldn't happen")
		return err
	} else {
		errDelete := r.client.Delete(context.TODO(),cephObjectStoreUser)
		if(errDelete != nil) {
			reqLogger.Info("Generic error for deleting cephObjectStoreUser, shouldn't happen")
			return errDelete
		}
	}

	cephObjectStore := &cephv1.CephObjectStore{}
	errLocal := getExactlyOneObject(r.client,cephObjectStore,m.ObjectMeta.Name,os.Getenv("ROOK_NAMESPACE"))
	if(errors.IsNotFound(errLocal)){
		reqLogger.Info("object store not created yet, we don't have to delete anything")
	} else if(errLocal!=nil) {
		reqLogger.Info("Generic error for getting ceph object store, shouldn't happen")
		return err
	} else{
		errDelete := r.client.Delete(context.TODO(),cephObjectStore)
		if(errDelete != nil) {
			reqLogger.Info("Generic error for deleting ceph object store, shouldn't happen")
			return errDelete
		}
	}

	configMapForRados := &corev1.ConfigMap{}
	errLocal = getExactlyOneObject(r.client,configMapForRados,"rook-ceph-rgw-"+m.ObjectMeta.Name+"-custom",os.Getenv("ROOK_NAMESPACE"))
	if(errors.IsNotFound(errLocal)){
		reqLogger.Info("configmap for rados not created yet, we don't have to delete anything")
	} else if(errLocal!=nil) {
		reqLogger.Info("Generic error for getting configmap for rados, shouldn't happen")
		return err
	} else{
		errDelete := r.client.Delete(context.TODO(),configMapForRados)
		if(errDelete != nil) {
			reqLogger.Info("Generic error for deleting configmap for rados, shouldn't happen")
			return errDelete
		}
	}

	reqLogger.Info("Successfully finalized dataset")
	return nil
}

func (r *ReconcileDataset) addFinalizer(reqLogger logr.Logger, m *comv1alpha1.Dataset) error {
	reqLogger.Info("Adding Finalizer for the Dataset")
	controllerutil.AddFinalizer(m, datasetsFinalizer)

	// Update CR
	err := r.client.Update(context.TODO(), m)
	if err != nil {
	reqLogger.Error(err, "Failed to update Dataset with finalizer")
	return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}