package repository

import (
	"context"
	"k8s.io/api/core/v1"
	v1Option "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sNode interface {
	GetNodeList(
		ctx context.Context,
		k8sClient kubernetes.Interface,
		option ...v1Option.ListOptions,
	) (*v1.NodeList, error)
}

type k8sNode struct {
}

func newK8sNode() K8sNode {
	return &k8sNode{}
}

func (n *k8sNode) GetNodeList(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	option ...v1Option.ListOptions,
) (*v1.NodeList, error) {
	reqOption := v1Option.ListOptions{}
	if len(option) > 0 {
		reqOption = option[0]
	}
	data, err := k8sClient.CoreV1().Nodes().List(
		ctx, reqOption,
	)
	if err != nil {
		return nil, err
	}
	return data, nil
}
