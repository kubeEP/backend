package repository

import (
	"context"
	v1 "k8s.io/api/apps/v1"
	v1Option "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sDeployment interface {
	GetDeployment(
		ctx context.Context,
		client kubernetes.Interface,
		namespace, name string,
		option ...v1Option.GetOptions,
	) (*v1.Deployment, error)
	GetAllDeployment(
		ctx context.Context,
		client kubernetes.Interface,
		namespace string,
		option ...v1Option.ListOptions,
	) (*v1.DeploymentList, error)
}

type k8sDeployment struct {
}

func newK8sDeployment() K8sDeployment {
	return &k8sDeployment{}
}

func (d *k8sDeployment) GetDeployment(
	ctx context.Context,
	client kubernetes.Interface,
	namespace, name string,
	option ...v1Option.GetOptions,
) (*v1.Deployment, error) {
	reqOption := v1Option.GetOptions{}
	if len(option) > 0 {
		reqOption = option[0]
	}
	return client.AppsV1().Deployments(namespace).Get(ctx, name, reqOption)
}

func (d *k8sDeployment) GetAllDeployment(
	ctx context.Context,
	client kubernetes.Interface,
	namespace string,
	option ...v1Option.ListOptions,
) (*v1.DeploymentList, error) {
	reqOption := v1Option.ListOptions{}
	if len(option) > 0 {
		reqOption = option[0]
	}
	return client.AppsV1().Deployments(namespace).List(ctx, reqOption)
}
