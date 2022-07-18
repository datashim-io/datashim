package test

import (
	"context"
	"testing"

	"github.com/datashim-io/datashim/src/apiclient/pkg/clientset/versioned/fake"
	"github.com/datashim-io/datashim/src/dataset-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFakeClient(t *testing.T) {

	client := fake.NewSimpleClientset()

	dataset := v1alpha1.Dataset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dataset1",
			Namespace: "default",
		},
		Spec: v1alpha1.DatasetSpec{
			Local: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			Type: "COS",
		},
		Status: v1alpha1.DatasetStatus{
			Caching: v1alpha1.DatasetStatusCondition{
				Status: "Disabled",
				Info:   "",
			},
			Provision: v1alpha1.DatasetStatusCondition{
				Status: "OK",
				Info:   "",
			},
		},
	}

	ctx := context.Background()

	_, err := client.ComV1alpha1().Datasets("default").Create(ctx, &dataset, metav1.CreateOptions{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	ds_list, err := client.ComV1alpha1().Datasets("default").List(ctx, metav1.ListOptions{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(ds_list.Items) != 1 {
		t.Errorf("Unexpected List size: %d", len(ds_list.Items))
	}
}
