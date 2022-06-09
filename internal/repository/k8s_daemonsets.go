package repository

import (
	"context"
	v1Apps "k8s.io/api/apps/v1"
	v1Option "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sDaemonSets interface {
	GetDaemonSetsList(
		ctx context.Context,
		client kubernetes.Interface,
		namespace string,
		option ...v1Option.ListOptions,
	) (*v1Apps.DaemonSetList, error)
}

type k8sDaemonSets struct {
}

func newK8sDaemonSets() K8sDaemonSets {
	return &k8sDaemonSets{}
}

func (k *k8sDaemonSets) GetDaemonSetsList(
	ctx context.Context,
	client kubernetes.Interface,
	namespace string,
	option ...v1Option.ListOptions,
) (*v1Apps.DaemonSetList, error) {
	reqOption := v1Option.ListOptions{}
	if len(option) > 0 {
		reqOption = option[0]
	}
	return client.AppsV1().DaemonSets(namespace).List(ctx, reqOption)
}
