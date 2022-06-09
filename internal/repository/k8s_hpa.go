package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant"
	errorConstant "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant/errors"
	v1Autoscale "k8s.io/api/autoscaling/v1"
	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/api/autoscaling/v2beta2"
	v1Core "k8s.io/api/core/v1"
	v1Option "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"
)

type K8sHPA interface {
	GetAllV1HPA(
		ctx context.Context,
		client kubernetes.Interface,
		namespace v1Core.Namespace,
		clusterID uuid.UUID,
	) ([]v1Autoscale.HorizontalPodAutoscaler, error)
	GetAllV2beta2HPA(
		ctx context.Context,
		client kubernetes.Interface,
		namespace v1Core.Namespace,
		clusterID uuid.UUID,
	) ([]v2beta2.HorizontalPodAutoscaler, error)
	GetAllV2beta1HPA(
		ctx context.Context,
		client kubernetes.Interface,
		namespace v1Core.Namespace,
		clusterID uuid.UUID,
	) ([]v2beta1.HorizontalPodAutoscaler, error)
	UpdateV2beta1HPA(
		ctx context.Context,
		client kubernetes.Interface,
		namespace string,
		clusterID uuid.UUID,
		hpa *v2beta1.HorizontalPodAutoscaler,
	) (*v2beta1.HorizontalPodAutoscaler, error)
	UpdateV1HPA(
		ctx context.Context,
		client kubernetes.Interface,
		namespace string,
		clusterID uuid.UUID,
		hpa *v1Autoscale.HorizontalPodAutoscaler,
	) (*v1Autoscale.HorizontalPodAutoscaler, error)
	UpdateV2beta2HPA(
		ctx context.Context,
		client kubernetes.Interface,
		namespace string,
		clusterID uuid.UUID,
		hpa *v2beta2.HorizontalPodAutoscaler,
	) (*v2beta2.HorizontalPodAutoscaler, error)
	GetV1HPA(
		ctx context.Context,
		client kubernetes.Interface,
		name string,
		namespace string,
		clusterID uuid.UUID,
	) (*v1Autoscale.HorizontalPodAutoscaler, error)
	GetV2beta1HPA(
		ctx context.Context,
		client kubernetes.Interface,
		name string,
		namespace string,
		clusterID uuid.UUID,
	) (*v2beta1.HorizontalPodAutoscaler, error)
	GetV2beta2HPA(
		ctx context.Context,
		client kubernetes.Interface,
		name string,
		namespace string,
		clusterID uuid.UUID,
	) (*v2beta2.HorizontalPodAutoscaler, error)
}

type k8sHPA struct {
	redisClient *redis.Client
}

func newK8sHPA(redisClient *redis.Client) K8sHPA {
	return &k8sHPA{
		redisClient: redisClient,
	}
}

const (
	HPACacheTime = 30 * time.Second
)

func (h *k8sHPA) GetAllV1HPA(
	ctx context.Context,
	client kubernetes.Interface,
	namespace v1Core.Namespace,
	clusterID uuid.UUID,
) ([]v1Autoscale.HorizontalPodAutoscaler, error) {
	key := fmt.Sprintf("hpa_v1_list_cluster_%s_ns_%s", clusterID, namespace.Name)
	if redisResponse := h.redisClient.Get(
		ctx,
		key,
	); redisResponse.Err() != nil {
		var HPAList v1Autoscale.HorizontalPodAutoscalerList
		b, err := redisResponse.Bytes()
		if err == nil {
			if string(b) == errorConstant.HPAListError {
				return nil, errors.New(errorConstant.HPAListError)
			}
			if err = HPAList.Unmarshal(b); err == nil {
				return HPAList.Items, nil
			}
		}
	}
	data, err := client.
		AutoscalingV1().
		HorizontalPodAutoscalers(namespace.Name).
		List(
			ctx,
			v1Option.ListOptions{},
		)
	if err != nil {
		_ = h.redisClient.Set(
			ctx,
			key,
			errorConstant.HPAListError,
			HPACacheTime,
		).Err()
		return nil, err
	}
	if b, err := data.Marshal(); err == nil {
		_ = h.redisClient.Set(ctx, key, b, HPACacheTime).Err()
	}
	return data.Items, nil
}

func (h *k8sHPA) GetAllV2beta2HPA(
	ctx context.Context,
	client kubernetes.Interface,
	namespace v1Core.Namespace,
	clusterID uuid.UUID,
) ([]v2beta2.HorizontalPodAutoscaler, error) {
	key := fmt.Sprintf("hpa_v2_beta_2_list_cluster_%s_ns_%s", clusterID, namespace.Name)
	if redisResponse := h.redisClient.Get(
		ctx,
		key,
	); redisResponse.Err() != nil {
		var HPAList v2beta2.HorizontalPodAutoscalerList
		b, err := redisResponse.Bytes()
		if err == nil {
			if string(b) == errorConstant.HPAListError {
				return nil, errors.New(errorConstant.HPAListError)
			}
			if err = HPAList.Unmarshal(b); err == nil {
				return HPAList.Items, nil
			}
		}
	}
	data, err := client.
		AutoscalingV2beta2().
		HorizontalPodAutoscalers(namespace.Name).
		List(
			ctx,
			v1Option.ListOptions{},
		)
	if err != nil {
		_ = h.redisClient.Set(
			ctx,
			key,
			errorConstant.HPAListError,
			HPACacheTime,
		).Err()
		return nil, err
	}
	if b, err := data.Marshal(); err == nil {
		_ = h.redisClient.Set(ctx, key, b, HPACacheTime).Err()
	}
	return data.Items, nil
}

func (h *k8sHPA) GetAllV2beta1HPA(
	ctx context.Context,
	client kubernetes.Interface,
	namespace v1Core.Namespace,
	clusterID uuid.UUID,
) ([]v2beta1.HorizontalPodAutoscaler, error) {
	key := fmt.Sprintf("hpa_v2_beta_1_list_cluster_%s_ns_%s", clusterID, namespace.Name)
	if redisResponse := h.redisClient.Get(
		ctx,
		key,
	); redisResponse.Err() != nil {
		var HPAList v2beta1.HorizontalPodAutoscalerList
		b, err := redisResponse.Bytes()
		if err == nil {
			if string(b) == errorConstant.HPAListError {
				return nil, errors.New(errorConstant.HPAListError)
			}
			if err = HPAList.Unmarshal(b); err == nil {
				return HPAList.Items, nil
			}
		}
	}
	data, err := client.
		AutoscalingV2beta1().
		HorizontalPodAutoscalers(namespace.Name).
		List(
			ctx,
			v1Option.ListOptions{},
		)
	if err != nil {
		_ = h.redisClient.Set(
			ctx,
			key,
			errorConstant.HPAListError,
			HPACacheTime,
		).Err()
		return nil, err
	}
	if b, err := data.Marshal(); err == nil {
		_ = h.redisClient.Set(ctx, key, b, HPACacheTime).Err()
	}
	return data.Items, nil
}

func (h *k8sHPA) UpdateV1HPA(
	ctx context.Context,
	client kubernetes.Interface,
	namespace string,
	clusterID uuid.UUID,
	hpa *v1Autoscale.HorizontalPodAutoscaler,
) (*v1Autoscale.HorizontalPodAutoscaler, error) {
	key := fmt.Sprintf("hpa_v1_list_cluster_%s_ns_%s", clusterID, namespace)
	data, err := client.
		AutoscalingV1().
		HorizontalPodAutoscalers(namespace).
		Update(
			ctx,
			hpa,
			v1Option.UpdateOptions{FieldManager: constant.K8sHPAUpdateFieldManager},
		)
	if err != nil {
		return nil, err
	}
	if b, err := data.Marshal(); err == nil {
		_ = h.redisClient.Set(ctx, key, b, HPACacheTime).Err()
	}
	return data, nil
}

func (h *k8sHPA) UpdateV2beta2HPA(
	ctx context.Context,
	client kubernetes.Interface,
	namespace string,
	clusterID uuid.UUID,
	hpa *v2beta2.HorizontalPodAutoscaler,
) (*v2beta2.HorizontalPodAutoscaler, error) {
	key := fmt.Sprintf("hpa_v2_beta_2_list_cluster_%s_ns_%s", clusterID, namespace)
	data, err := client.
		AutoscalingV2beta2().
		HorizontalPodAutoscalers(namespace).
		Update(
			ctx,
			hpa,
			v1Option.UpdateOptions{
				FieldManager: constant.K8sHPAUpdateFieldManager,
			},
		)
	if err != nil {
		return nil, err
	}
	if b, err := data.Marshal(); err == nil {
		_ = h.redisClient.Set(ctx, key, b, HPACacheTime).Err()
	}
	return data, nil
}

func (h *k8sHPA) UpdateV2beta1HPA(
	ctx context.Context,
	client kubernetes.Interface,
	namespace string,
	clusterID uuid.UUID,
	hpa *v2beta1.HorizontalPodAutoscaler,
) (*v2beta1.HorizontalPodAutoscaler, error) {
	key := fmt.Sprintf("hpa_v2_beta_1_list_cluster_%s_ns_%s", clusterID, namespace)
	data, err := client.
		AutoscalingV2beta1().
		HorizontalPodAutoscalers(namespace).
		Update(
			ctx,
			hpa,
			v1Option.UpdateOptions{
				FieldManager: constant.K8sHPAUpdateFieldManager,
			},
		)
	if err != nil {
		return nil, err
	}
	if b, err := data.Marshal(); err == nil {
		_ = h.redisClient.Set(ctx, key, b, HPACacheTime).Err()
	}
	return data, nil
}

func (h *k8sHPA) GetV1HPA(
	ctx context.Context,
	client kubernetes.Interface,
	name string,
	namespace string,
	clusterID uuid.UUID,
) (*v1Autoscale.HorizontalPodAutoscaler, error) {
	key := fmt.Sprintf("hpa_v1_cluster_%s_ns_%s_name_%s", clusterID, namespace, name)
	if redisResponse := h.redisClient.Get(
		ctx,
		key,
	); redisResponse.Err() != nil {
		var hpa *v1Autoscale.HorizontalPodAutoscaler
		b, err := redisResponse.Bytes()
		if err == nil {
			if string(b) == errorConstant.HPAError {
				return nil, errors.New(string(b))
			}
			if err = hpa.Unmarshal(b); err == nil {
				return hpa, nil
			}
		}
	}
	data, err := client.
		AutoscalingV1().
		HorizontalPodAutoscalers(namespace).
		Get(
			ctx,
			name,
			v1Option.GetOptions{},
		)
	if err != nil {
		_ = h.redisClient.Set(ctx, key, errorConstant.HPAError, HPACacheTime).Err()
		return nil, err
	}
	if b, err := data.Marshal(); err == nil {
		_ = h.redisClient.Set(ctx, key, b, HPACacheTime).Err()
	}
	return data, nil
}

func (h *k8sHPA) GetV2beta1HPA(
	ctx context.Context,
	client kubernetes.Interface,
	name string,
	namespace string,
	clusterID uuid.UUID,
) (*v2beta1.HorizontalPodAutoscaler, error) {
	key := fmt.Sprintf("hpa_v2beta1_cluster_%s_ns_%s_name_%s", clusterID, namespace, name)
	if redisResponse := h.redisClient.Get(
		ctx,
		key,
	); redisResponse.Err() != nil {
		var hpa *v2beta1.HorizontalPodAutoscaler
		b, err := redisResponse.Bytes()
		if err == nil {
			if string(b) == errorConstant.HPAError {
				return nil, errors.New(string(b))
			}
			if err = hpa.Unmarshal(b); err == nil {
				return hpa, nil
			}
		}
	}
	data, err := client.
		AutoscalingV2beta1().
		HorizontalPodAutoscalers(namespace).
		Get(
			ctx,
			name,
			v1Option.GetOptions{},
		)
	if err != nil {
		_ = h.redisClient.Set(ctx, key, errorConstant.HPAError, HPACacheTime).Err()
		return nil, err
	}
	if b, err := data.Marshal(); err == nil {
		_ = h.redisClient.Set(ctx, key, b, HPACacheTime).Err()
	}
	return data, nil
}

func (h *k8sHPA) GetV2beta2HPA(
	ctx context.Context,
	client kubernetes.Interface,
	name string,
	namespace string,
	clusterID uuid.UUID,
) (*v2beta2.HorizontalPodAutoscaler, error) {
	key := fmt.Sprintf("hpa_v2beta2_cluster_%s_ns_%s_name_%s", clusterID, namespace, name)
	if redisResponse := h.redisClient.Get(
		ctx,
		key,
	); redisResponse.Err() != nil {
		var hpa *v2beta2.HorizontalPodAutoscaler
		b, err := redisResponse.Bytes()
		if err == nil {
			if string(b) == errorConstant.HPAError {
				return nil, errors.New(string(b))
			}
			if err = hpa.Unmarshal(b); err == nil {
				return hpa, nil
			}
		}
	}
	data, err := client.
		AutoscalingV2beta2().
		HorizontalPodAutoscalers(namespace).
		Get(
			ctx,
			name,
			v1Option.GetOptions{},
		)
	if err != nil {
		_ = h.redisClient.Set(ctx, key, errorConstant.HPAError, HPACacheTime).Err()
		return nil, err
	}
	if b, err := data.Marshal(); err == nil {
		_ = h.redisClient.Set(ctx, key, b, HPACacheTime).Err()
	}
	return data, nil
}
