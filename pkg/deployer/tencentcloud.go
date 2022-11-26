package deployer

import (
	"fmt"
	cdn "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdn/v20180606"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"log"
	"os"
)

type TencentCloudDeployer struct {
	client     *cdn.Client
	updateOnly bool
}

func (*TencentCloudDeployer) Name() string {
	return "tencentcloud"
}

// Deploy deploys cert and key to all related domains, while domains indicate the domains contains in certificate
func (d *TencentCloudDeployer) Deploy(domains []string, cert, key string) error {
	if len(domains) < 1 {
		return nil
	}

	log.Println("getting tencent cloud CDN domains matching given certificates")
	for _, domain := range domains {
		normalizedDomain := normalizeDomain(domain)
		fuzzy := false
		if normalizedDomain[0] == '.' {
			fuzzy = true
		}
		var pageNumber int64 = 1
		var pageSize int64 = 1000
		for true {
			log.Printf("domain %s, page %d ...", normalizedDomain, pageNumber)
			request := cdn.NewDescribeDomainsConfigRequest()
			request.Limit = common.Int64Ptr(pageSize)
			request.Offset = common.Int64Ptr((pageNumber - 1) * pageSize)
			request.Filters = []*cdn.DomainFilter{
				&cdn.DomainFilter{
					Name:  common.StringPtr("domain"),
					Value: common.StringPtrs([]string{normalizedDomain}),
					Fuzzy: common.BoolPtr(fuzzy),
				},
			}
			cdnDomains, err := d.client.DescribeDomainsConfig(request)
			if err != nil {
				return fmt.Errorf("failed to describe user domains with suffix %s: %w", normalizedDomain, err)
			}
			for _, cdnDomain := range cdnDomains.Response.Domains {
				if d.checkDomainDeploy(cdnDomain) {
					err := d.deployCert(cdnDomain, cert, key)
					if err != nil {
						return fmt.Errorf("failed to deploy domain %s: %w", *cdnDomain.Domain, err)
					}
				}
			}
			if *cdnDomains.Response.TotalNumber > (pageSize * pageNumber) {
				pageNumber = pageNumber + 1
			} else {
				break
			}
		}
	}

	return nil
}

func (d *TencentCloudDeployer) checkDomainDeploy(cdnDomain *cdn.DetailDomain) bool {
	if cdnDomain.Domain == nil || cdnDomain.Https == nil || cdnDomain.Status == nil {
		return false
	}
	if *cdnDomain.Status != "online" && *cdnDomain.Status != "processing" {
		return false
	}
	if d.updateOnly {
		return cdnDomain.Https.Switch != nil && *cdnDomain.Https.Switch == "on"
	} else {
		return true
	}
}

func (d *TencentCloudDeployer) deployCert(cdnDomain *cdn.DetailDomain, cert string, key string) error {
	log.Printf("deploying cert for domain: %s", *cdnDomain.Domain)
	if cdnDomain.Https == nil {
		cdnDomain.Https = &cdn.Https{
			Switch: common.StringPtr("on"),
		}
	}
	if cdnDomain.Https.CertInfo == nil {
		cdnDomain.Https.CertInfo = &cdn.ServerCert{}
	}
	cdnDomain.Https.Switch = common.StringPtr("on")
	cdnDomain.Https.CertInfo.Certificate = common.StringPtr(cert)
	cdnDomain.Https.CertInfo.PrivateKey = common.StringPtr(key)

	request := cdn.NewUpdateDomainConfigRequest()
	request.Domain = cdnDomain.Domain
	request.Https = cdnDomain.Https

	_, err := d.client.UpdateDomainConfig(request)
	if err != nil {
		return fmt.Errorf("failed to call update domain api: %w", err)
	}
	return nil
}

var _ Deployer = (*TencentCloudDeployer)(nil)

func CreateTencentCloudDeployer() (*TencentCloudDeployer, error) {
	credentials := common.NewCredential(
		os.Getenv("TENCENTCLOUD_SECRET_ID"),
		os.Getenv("TENCENTCLOUD_SECRET_KEY"),
	)
	cpf := profile.NewClientProfile()

	client, err := cdn.NewClient(credentials, "", cpf)
	if err != nil {
		return nil, fmt.Errorf("failed to create tencent cloud sdk instance: %w", err)
	}

	deployer := TencentCloudDeployer{
		client:     client,
		updateOnly: os.Getenv("TENCENTCLOUD_CERT_UPDATE_ONLY") == "true",
	}

	return &deployer, nil
}
