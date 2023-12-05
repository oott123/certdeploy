package deployer

import (
	"encoding/base64"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azcertificates"
	"github.com/oott123/certdeploy/pkg/certparser"
	"github.com/oott123/certdeploy/pkg/util"
	"golang.org/x/exp/slices"
	"golang.org/x/net/context"
	"log"
	"os"
)

type AzureDeployer struct {
	client *azcertificates.Client
}

var _ Deployer = (*AzureDeployer)(nil)

func (*AzureDeployer) Name() string {
	return "azure"
}

// Deploy deploys cert and key to all related domains, while domains indicate the domains contains in certificate
func (d *AzureDeployer) Deploy(domains []string, cert, key string) error {
	log.Printf("finding certificates in keyvault to deploy")
	certsDomainsMap, err := d.getCertificatesDomainsMap()
	if err != nil {
		return fmt.Errorf("failed to get certs domains map: %w", err)
	}
	found := false
	for name, domainsInService := range *certsDomainsMap {
		for _, domainInService := range domainsInService {
			if !slices.Contains(domains, domainInService) {
				continue
			}
		}
		found = true
		log.Printf("importing certificate to update %s", name)
		err = d.importCertificate(name, cert, key)
		if err != nil {
			return fmt.Errorf("failed to import certificate: %w", err)
		}
	}
	if !found {
		log.Printf("unable to find certificates in keyvault to deploy")
	}
	return nil
}

func (d *AzureDeployer) importCertificate(name, cert, key string) error {
	pfx, err := util.PemToPfx(cert, key)
	if err != nil {
		return fmt.Errorf("failed to convert pem to pfx: %w", err)
	}
	encodedCertificate := base64.StdEncoding.EncodeToString(pfx)
	contentType := "application/x-pkcs12"
	_, err = d.client.ImportCertificate(context.Background(), name, azcertificates.ImportCertificateParameters{
		Base64EncodedCertificate: &encodedCertificate,
		CertificateAttributes:    nil,
		CertificatePolicy: &azcertificates.CertificatePolicy{
			Attributes:       nil,
			IssuerParameters: nil,
			KeyProperties:    nil,
			LifetimeActions:  nil,
			SecretProperties: &azcertificates.SecretProperties{
				ContentType: &contentType,
			},
			X509CertificateProperties: nil,
			ID:                        nil,
		},
		Password: nil,
		Tags:     nil,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to volcRequest import certificate to %s: %w", name, err)
	}
	return nil
}

func (d *AzureDeployer) getCertificatesDomainsMap() (*map[string][]string, error) {
	pager := d.client.NewListCertificatesPager(nil)
	certsDomainsMap := make(map[string][]string)
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to naviate to next page: %w", err)
		}
		for _, cert := range page.Value {
			name := cert.ID.Name()
			if _, found := certsDomainsMap[name]; found {
				continue
			}
			certDetails, err := d.client.GetCertificate(context.Background(), name, cert.ID.Version(), nil)
			if err != nil {
				return nil, fmt.Errorf("failed to get certificate %s: %w", name, err)
			}

			pem := fmt.Sprintf("-----BEGIN CERTIFICATE-----\n%s\n-----END CERTIFICATE-----", base64.StdEncoding.EncodeToString(certDetails.CER))
			domains, err := certparser.DomainsFromCert(pem)
			if err != nil {
				return nil, fmt.Errorf("failed to parse certificate %s: %w", name, err)
			}
			certsDomainsMap[name] = domains
		}
	}
	return &certsDomainsMap, nil
}

func CreateAzureDeployer() (*AzureDeployer, error) {
	keyVaultUri := os.Getenv("AZURE_KEY_VAULT_URI")
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get azure credentials: %w", err)
	}
	client, err := azcertificates.NewClient(keyVaultUri, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure certificate client: %w", err)
	}
	deployer := AzureDeployer{
		client: client,
	}
	return &deployer, nil
}
