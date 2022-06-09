package k8sClient

import (
	"encoding/base64"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type Credentials struct {
	Certificate        string
	Name               string
	ServerEndpoint     string
	AuthProviderConfig *api.AuthProviderConfig
}

func GetClient(credentials *Credentials) (*kubernetes.Clientset, error) {
	cert, err := base64.StdEncoding.DecodeString(credentials.Certificate)
	if err != nil {
		return nil, err
	}

	kubernetesConfig := api.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters:   map[string]*api.Cluster{},
		AuthInfos:  map[string]*api.AuthInfo{},
		Contexts:   map[string]*api.Context{},
	}

	name := credentials.Name

	kubernetesConfig.Clusters[name] = &api.Cluster{
		CertificateAuthorityData: cert,
		Server:                   credentials.ServerEndpoint,
	}

	kubernetesConfig.Contexts[name] = &api.Context{
		Cluster:  name,
		AuthInfo: name,
	}

	kubernetesConfig.AuthInfos[name] = &api.AuthInfo{
		AuthProvider: credentials.AuthProviderConfig,
	}

	cfg, err := clientcmd.
		NewNonInteractiveClientConfig(
			kubernetesConfig,
			name,
			&clientcmd.ConfigOverrides{CurrentContext: name},
			nil,
		).ClientConfig()
	if err != nil {
		return nil, err
	}

	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return k8sClient, nil
}
