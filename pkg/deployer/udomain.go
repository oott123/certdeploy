package deployer

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/oott123/certdeploy/pkg/util"
	"log"
	"os"
	"time"
)

type UDomainDeployer struct {
	apiKey string
}

type getSubDomainResult struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Payload []struct {
		CustomerID           string `json:"customerID"`
		OrderID              string `json:"orderID"`
		SubdomainCDNType     string `json:"subdomainCDNType"`
		SubdomainCNAME       string `json:"subdomainCNAME"`
		SubdomainCNAMEStatus string `json:"subdomainCNAMEStatus"`
		SubdomainID          int    `json:"subdomainID"`
		SubdomainName        string `json:"subdomainName"`
		SubdomainStatus      string `json:"subdomainStatus"`
		CreateDate           string `json:"createDate"`
		CreatedBy            string `json:"createdBy"`
		UpdateBy             string `json:"updateBy"`
		UpdateDate           string `json:"updateDate"`
	} `json:"payload"`
}

type postCertificateResult struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Payload struct {
		CertificateID   int    `json:"certificateID"`
		CertificateName string `json:"certificateName"`
		CreateDate      string `json:"createDate"`
		CreatedBy       string `json:"createdBy"`
		PrivateKey      string `json:"privateKey"`
		PublicKey       string `json:"publicKey"`
	} `json:"payload"`
}

type postConfigurationResult struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type postConfigurationRequest struct {
	ConfigCategory string `json:"configCategory"`
	ConfigItem     string `json:"configItem"`
	ConfigValue    struct {
		CertificateID int `json:"certificateID"`
	} `json:"configValue"`
	SubdomainID int `json:"subdomainID"`
}

type postCertificateRequest struct {
	CertificateName string `json:"certificateName"`
	PrivateKey      string `json:"privateKey"`
	PublicKey       string `json:"publicKey"`
}

var _ Deployer = (*UDomainDeployer)(nil)

func (*UDomainDeployer) Name() string {
	return "udomain"
}

// Deploy deploys cert and key to all related domains, while domains indicate the domains contains in certificate
func (d *UDomainDeployer) Deploy(domains []string, cert, key string) error {
	c := resty.New().SetHeader("Authorization", d.apiKey).SetBaseURL("https://cdn.8338.hk/api")

	response := getSubDomainResult{
		Code: "failed",
	}
	_, err := c.R().SetResult(&response).SetError(&response).Get("/c/v1/subdomain")
	if err != nil {
		return fmt.Errorf("failed to volcRequest domain: %w", err)
	}
	if response.Code != "0" {
		return fmt.Errorf("failed to get domain %s(%s)", response.Code, response.Message)
	}

	subdomainIds := make([]int, 0)
	for _, subdomain := range response.Payload {
		for _, domainInCert := range domains {
			if util.MatchDomain(domainInCert, subdomain.SubdomainName) &&
				(subdomain.SubdomainStatus == "ACTIVE" || subdomain.SubdomainStatus == "PROCESSING") {
				log.Printf("queued to update domain %s(#%d)", subdomain.SubdomainName, subdomain.SubdomainID)
				subdomainIds = append(subdomainIds, subdomain.SubdomainID)
				break
			}
		}
	}

	if len(subdomainIds) <= 0 {
		log.Printf("unable to find domains suited for certificate")
		return nil
	}

	// upload certificate
	certRequest := postCertificateRequest{
		CertificateName: fmt.Sprintf("%s(%s)", domains[0], time.Now().UTC().Format("2006-01-02")),
		PrivateKey:      key,
		PublicKey:       cert,
	}
	certResult := postCertificateResult{
		Code: "failed",
	}
	_, err = c.R().SetResult(&certResult).SetError(&certResult).SetBody(&certRequest).Post("/c/v1/certificate")
	if err != nil {
		return fmt.Errorf("failed to upload certificate volcRequest: %w", err)
	}
	if certResult.Code != "0" {
		return fmt.Errorf("failed to upload certificate %s(%s)", certResult.Code, certResult.Message)
	}
	certId := certResult.Payload.CertificateID
	log.Printf("successfully uploaded certificate #%d", certId)
	// apply certificate
	for _, subdomainId := range subdomainIds {
		request := postConfigurationRequest{
			ConfigCategory: "HTTPS",
			ConfigItem:     "CERTIFICATE",
			SubdomainID:    subdomainId,
			ConfigValue: struct {
				CertificateID int `json:"certificateID"`
			}{CertificateID: certId},
		}
		result := postConfigurationResult{
			Code: "failed",
		}
		r, err := c.R().SetBody(&request).SetError(&result).Put("/c/v1/configuration")
		if err != nil {
			return fmt.Errorf("failed to update domain volcRequest: %w", err)
		}
		if r.StatusCode() > 299 {
			return fmt.Errorf("failed to update domain: %s %s", result.Code, result.Message)
		}
		log.Printf("successfully updated domain #%d", subdomainId)
	}
	return nil
}

func CreateUDomainDeployer() (*UDomainDeployer, error) {
	deployer := UDomainDeployer{apiKey: os.Getenv("UDOMAIN_API_KEY")}
	return &deployer, nil
}
