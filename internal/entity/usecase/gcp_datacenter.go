package UCEntity

type GCPSAKeyCredentials struct {
	Type                    *string `json:"type" validate:"required"`
	ProjectId               *string `json:"project_id" validate:"required"`
	PrivateKeyId            *string `json:"private_key_id" validate:"required"`
	PrivateKey              *string `json:"private_key" validate:"required"`
	ClientEmail             *string `json:"client_email" validate:"required"`
	ClientId                *string `json:"client_id" validate:"required"`
	AuthUri                 *string `json:"auth_uri" validate:"required"`
	TokenUri                *string `json:"token_uri" validate:"required"`
	AuthProviderX509CertUrl *string `json:"auth_provider_x509_cert_url" validate:"required"`
	ClientX509CertUrl       *string `json:"client_x509_cert_url" validate:"required"`
}

type GCPDatacenterMetaData struct {
	ProjectId string `json:"project_id"`
	SAEmail   string `json:"sa_email"`
}
