package deployer

import (
	"fmt"
	resty "github.com/go-resty/resty/v2"
	gjson "github.com/tidwall/gjson"
	"golang.org/x/net/publicsuffix"
	"log"
	"net/http/cookiejar"
	"os"
)

type UpyunDeployer struct {
	username string
	password string
	jar      *cookiejar.Jar
	client   *resty.Client
}

func (u *UpyunDeployer) Name() string {
	return "upyun"
}

func (u *UpyunDeployer) Deploy(_ []string, cert, key string) error {
	log.Println("upyun logging in")
	err := u.Login()
	if err != nil {
		return fmt.Errorf("upyun login failed: %w", err)
	}

	log.Println("upyun uploading certificate")
	certId, err := u.UploadCertificate(cert, key)
	if err != nil {
		return fmt.Errorf("upyun upload cert failed: %w", err)
	}

	log.Printf("upyun certificate id: %s, getting domains", certId)
	domains, err := u.DomainsByCertificate(certId)
	if err != nil {
		return fmt.Errorf("upyun get domains failed: %w", err)
	}

	for _, domain := range domains {
		log.Printf("deploing certificate for domain: %s", domain)
		err = u.SetDomainCertificate(certId, domain)
		if err != nil {
			return fmt.Errorf("upyun set domain certificate failed: %w", err)
		}
	}

	return nil
}

func (u *UpyunDeployer) Login() error {
	resp, err := u.client.R().SetBody(map[string]string{
		"username": u.username,
		"password": u.password,
	}).Post("https://console.upyun.com/accounts/signin/")

	if err = checkApiResult(resp, err); err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	return nil
}

func (u *UpyunDeployer) UploadCertificate(cert, key string) (string, error) {
	resp, err := u.client.R().SetBody(map[string]string{
		"certificate": cert,
		"private_key": key,
	}).Post("https://console.upyun.com/api/https/certificate/")

	if err = checkApiResult(resp, err); err != nil {
		return "", fmt.Errorf("failed to upload: %w", err)
	}

	return gjson.Get(resp.String(), "data.result.certificate_id").String(), nil
}

func (u *UpyunDeployer) DomainsByCertificate(certId string) ([]string, error) {
	resp, err := u.client.R().Get("https://console.upyun.com/api/https/certificate/manager/?certificate_id=" + certId)

	if err = checkApiResult(resp, err); err != nil {
		return nil, fmt.Errorf("failed to get domains: %w", err)
	}

	domains := make([]string, 0)
	list := gjson.Get(resp.String(), "data.domains.#.name")
	for _, item := range list.Array() {
		domains = append(domains, item.String())
	}

	return domains, nil
}

func (u *UpyunDeployer) SetDomainCertificate(certId string, domain string) error {
	resp, err := u.client.R().SetBody(map[string]interface{}{
		"certificate_id": certId,
		"domain":         domain,
		"https":          true,
	}).Post("https://console.upyun.com/api/https/certificate/manager/")

	if err = checkApiResult(resp, err); err != nil {
		if gjson.Get(resp.String(), "data.error_code").String() == "21713" {
			return u.MigrateDomainCertificate(certId, domain)
		}
		return fmt.Errorf("failed to set https: %w", err)
	}

	return nil
}

func (u *UpyunDeployer) MigrateDomainCertificate(certId, domain string) error {
	resp, err := u.client.R().SetBody(map[string]string{
		"crt_id":      certId,
		"domain_name": domain,
	}).Post("https://console.upyun.com/api/https/migrate/domain")

	if err = checkApiResult(resp, err); err != nil {
		return fmt.Errorf("failed to migrate domain: %w", err)
	}

	return nil
}

func checkApiResult(resp *resty.Response, err error) error {
	if err != nil {
		return fmt.Errorf("volcRequest failed: %w", err)
	}
	if resp == nil {
		return fmt.Errorf("volcRequest failed: response is nil")
	}
	json := resp.String()
	if gjson.Get(json, "data.error_code").Exists() {
		return fmt.Errorf("%s", gjson.Get(json, "data.message"))
	}
	if gjson.Get(json, "error_code").Exists() {
		return fmt.Errorf("%s(%s)", gjson.Get(json, "error"), gjson.Get(json, "message"))
	}
	return nil
}

var _ Deployer = (*UpyunDeployer)(nil)

func CreateUpyunDeployer() (*UpyunDeployer, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})

	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar for upyun: %w", err)
	}

	client := resty.New()
	client.SetCookieJar(jar)

	return &UpyunDeployer{
		username: os.Getenv("UPYUN_USERNAME"),
		password: os.Getenv("UPYUN_PASSWORD"),
		jar:      jar,
		client:   client,
	}, nil
}
