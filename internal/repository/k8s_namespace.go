package repository

import (
	"context"
	v1 "k8s.io/api/core/v1"
	v1Option "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sNamespace interface {
	GetAllNamespace(ctx context.Context, client kubernetes.Interface) ([]v1.Namespace, error)
}

type k8sNamespace struct {
}

func newK8sNamespace() K8sNamespace {
	return &k8sNamespace{}
}

func (n *k8sNamespace) GetAllNamespace(ctx context.Context, client kubernetes.Interface) ([]v1.Namespace, error) {
	data, err := client.
		CoreV1().
		Namespaces().
		List(
			ctx,
			v1Option.ListOptions{},
		)
	if err != nil {
		return nil, err
	}
	return data.Items, nil
}
