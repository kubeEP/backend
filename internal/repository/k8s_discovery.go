package repository

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8SDiscovery interface {
	GetServerGroups(k8sClient kubernetes.Interface) (*v1.APIGroupList, error)
}

type k8sDiscovery struct {
}

func newK8sDiscovery() K8SDiscovery {
	return &k8sDiscovery{}
}

func (k *k8sDiscovery) GetServerGroups(k8sClient kubernetes.Interface) (*v1.APIGroupList, error) {
	return k8sClient.Discovery().ServerGroups()
}
