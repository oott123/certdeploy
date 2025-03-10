package deployer

import (
	"fmt"
	"log"
	"os"
	"strings"

	cdn "github.com/alibabacloud-go/cdn-20180510/v5/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
)

type AliyunDeployer struct {
	client        *cdn.Client
	updateOnly    bool
	resourceGroup string
}

func (*AliyunDeployer) Name() string {
	return "aliyun"
}

// Deploy deploys cert and key to all related domains, while domains indicate the domains contains in certificate
func (d *AliyunDeployer) Deploy(domains []string, cert, key string) error {
	if len(domains) < 1 {
		return nil
	}

	log.Println("getting aliyun CDN domains matching given certificates")
	domainsToDeploy := make(map[string]bool)
	for _, domain := range domains {
		normalizedDomain := normalizeWildcardDomain(domain)
		matchType := "full_match"
		if normalizedDomain[0] == '.' {
			matchType = "suf_match"
		}
		var pageNumber int32 = 1
		for true {
			log.Printf("domain %s %s, page %d ...", normalizedDomain, matchType, pageNumber)
			request := cdn.DescribeUserDomainsRequest{
				DomainName:       tea.String(normalizedDomain),
				DomainSearchType: tea.String(matchType),
				CheckDomainShow:  tea.Bool(false),
				PageNumber:       tea.Int32(pageNumber),
			}
			if d.resourceGroup != "" {
				request.ResourceGroupId = tea.String(d.resourceGroup)
			}
			cdnDomains, err := d.client.DescribeUserDomains(&request)
			if err != nil {
				return fmt.Errorf("failed to describe user domains with suffix %s: %w", normalizedDomain, err)
			}
			for _, cdnDomain := range cdnDomains.Body.Domains.PageData {
				if cdnDomain.DomainName != nil && d.checkDomainStatus(cdnDomain.DomainStatus) {
					if d.updateOnly {
						if *cdnDomain.SslProtocol == "on" {
							domainsToDeploy[*cdnDomain.DomainName] = true
						}
					} else {
						domainsToDeploy[*cdnDomain.DomainName] = true
					}
				}
			}
			if *cdnDomains.Body.TotalCount > (*cdnDomains.Body.PageSize * (*cdnDomains.Body.PageNumber)) {
				pageNumber = int32(*(cdnDomains.Body.PageNumber) + 1)
			} else {
				break
			}
		}
	}

	log.Printf("got %d domains to deploy", len(domainsToDeploy))

	i := 0
	domainsChunk := make([]string, 0)
	for domain := range domainsToDeploy {
		i++
		domainsChunk = append(domainsChunk, domain)
		if i >= 50 {
			err := d.deployCert(domainsChunk, normalizeWildcardDomain(domains[0]), cert, key)
			if err != nil {
				return fmt.Errorf("failed to deploy cert: %w", err)
			}
			i = 0
			domainsChunk = make([]string, 0)
		}
	}
	if len(domainsChunk) > 0 {
		err := d.deployCert(domainsChunk, normalizeWildcardDomain(domains[0]), cert, key)
		if err != nil {
			return fmt.Errorf("failed to deploy cert: %w", err)
		}
	}
	return nil
}

func (d *AliyunDeployer) checkDomainStatus(status *string) bool {
	return *status == "online" || *status == "configuring"
}

func (d *AliyunDeployer) deployCert(cdnDomains []string, name, cert, key string) error {
	for i, domain := range cdnDomains {
		log.Printf("deploying cert for domain %s (%d of %d)", domain, i+1, len(cdnDomains))
		request := cdn.SetCdnDomainSSLCertificateRequest{
			DomainName:  tea.String(domain),
			CertType:    tea.String("upload"),
			SSLPub:      tea.String(cert),
			SSLPri:      tea.String(key),
			SSLProtocol: tea.String("on"),
		}
		_, err := d.client.SetCdnDomainSSLCertificate(&request)
		if err != nil {
			return fmt.Errorf("failed to call set cert api: %w", err)
		}
	}

	return nil
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func normalizeWildcardDomain(domain string) string {
	if strings.Index(domain, "*") == 0 {
		return strings.ToLower(domain[1:])
	} else {
		return strings.ToLower(domain)
	}
}

var _ Deployer = (*AliyunDeployer)(nil)

func CreateAliyunDeployer() (*AliyunDeployer, error) {
	config := openapi.Config{
		AccessKeyId:     tea.String(os.Getenv("ALIYUN_ACCESS_KEY_ID")),
		AccessKeySecret: tea.String(os.Getenv("ALIYUN_ACCESS_KEY_SECRET")),
	}

	client, err := cdn.NewClient(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to create aliyun sdk instance: %w", err)
	}

	deployer := AliyunDeployer{
		client:        client,
		updateOnly:    os.Getenv("ALIYUN_CERT_UPDATE_ONLY") == "true",
		resourceGroup: os.Getenv("ALIYUN_CERT_RESOURCE_GROUP"),
	}

	return &deployer, nil
}
