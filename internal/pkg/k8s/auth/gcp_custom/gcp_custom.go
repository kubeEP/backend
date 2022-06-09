package gcpCustomAuth

import (
	"errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"k8s.io/apimachinery/pkg/util/net"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"net/http"
	"sync"
)

const (
	CredentialsNameConfigKey = "credentials_name"
	AuthName                 = "gcp_custom"
)

var (
	lock sync.Mutex

	credentialList = make(map[string]*google.Credentials)
)

func RegisterGoogleCredentials(credentialsName string, credential *google.Credentials) {
	lock.Lock()
	defer lock.Unlock()

	credentialList[credentialsName] = credential
}

func RegisterK8SGCPCustomAuthProvider() {
	if err := restclient.RegisterAuthProviderPlugin(
		AuthName,
		newGCPCustomAuthProvider,
	); err != nil {
		klog.Fatalf("Failed to register gcp_custom auth plugin: %v", err)
	}
}

type gcpCustomAuthProvider struct {
	tokenSource oauth2.TokenSource
	persister   restclient.AuthProviderConfigPersister
}

func newGCPCustomAuthProvider(
	_ string,
	gcpConfig map[string]string,
	persister restclient.AuthProviderConfigPersister,
) (restclient.AuthProvider, error) {
	lock.Lock()
	defer lock.Unlock()

	credentialsName := gcpConfig[CredentialsNameConfigKey]
	credentials, ok := credentialList[credentialsName]
	if !ok {
		return nil, errors.New("credentials not found")
	}
	return &gcpCustomAuthProvider{credentials.TokenSource, persister}, nil
}

func (g *gcpCustomAuthProvider) WrapTransport(rt http.RoundTripper) http.RoundTripper {
	return &conditionalTransport{
		&oauth2.Transport{Source: g.tokenSource, Base: rt},
		g.persister,
		make(map[string]string),
	}
}

func (g *gcpCustomAuthProvider) Login() error { return nil }

type conditionalTransport struct {
	oauthTransport *oauth2.Transport
	persister      restclient.AuthProviderConfigPersister
	resetCache     map[string]string
}

var _ net.RoundTripperWrapper = &conditionalTransport{}

func (t *conditionalTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(req.Header.Get("Authorization")) != 0 {
		return t.oauthTransport.Base.RoundTrip(req)
	}

	res, err := t.oauthTransport.RoundTrip(req)

	if err != nil {
		return nil, err
	}

	if res.StatusCode == 401 {
		klog.V(4).Infof("The credentials that were supplied are invalid for the target cluster")
		t.persister.Persist(t.resetCache)
	}

	return res, nil
}

func (t *conditionalTransport) WrappedRoundTripper() http.RoundTripper { return t.oauthTransport.Base }
