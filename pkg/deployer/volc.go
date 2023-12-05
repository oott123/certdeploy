package deployer

import (
	"encoding/json"
	"fmt"
	volcBase "github.com/volcengine/volc-sdk-golang/base"
	"github.com/volcengine/volc-sdk-golang/service/cdn"
	"golang.org/x/exp/slices"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type VolcDeployer struct {
	cCdn  *cdn.CDN
	cDcdn *volcBase.Client
}

func (v *VolcDeployer) Name() string {
	return "volc"
}

func (v *VolcDeployer) Deploy(certDomains []string, cert, key string) error {
	err, certId := v.uploadCertificate(cert, key)
	if err != nil {
		return err
	}

	targetStr := os.Getenv("VOLC_DEPLOY_TARGETS")
	if targetStr == "" {
		targetStr = "cdn,dcdn"
	}
	targets := strings.Split(targetStr, ",")

	if slices.Contains(targets, "cdn") {
		err = v.deployCdn(certId)
		if err != nil {
			return err
		}
	}

	if slices.Contains(targets, "dcdn") {
		err = v.deployDcdn(certDomains, certId)
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *VolcDeployer) uploadCertificate(cert string, key string) (error, string) {
	certResp, err := v.cCdn.AddCdnCertificate(&cdn.AddCdnCertificateRequest{
		Certificate: cdn.Certificate{
			Certificate: cdn.GetStrPtr(cert),
			PrivateKey:  cdn.GetStrPtr(key),
		},
		CertInfo: &cdn.AddCdnCertInfo{
			Desc: cdn.GetStrPtr(time.Now().Format("certdeploy-20060102")),
		},
		Source: cdn.GetStrPtr("volc_cert_center"),
	})
	if err != nil {
		return fmt.Errorf("create volc cert: %w", err), ""
	}

	certId := certResp.Result
	log.Printf("uploaded cert id %s", certId)
	return nil, certId
}

func (v *VolcDeployer) deployCdn(certId string) error {
	var err error
	configResp, err := v.cCdn.DescribeCertConfig(&cdn.DescribeCertConfigRequest{
		CertId: certId,
		Status: cdn.GetStrPtr("configuring,online"),
	})
	if err != nil {
		return fmt.Errorf("describe volc cert %s: %w", certId, err)
	}

	domains := make([]string, 0)
	for _, dom := range configResp.Result.CertNotConfig {
		domains = append(domains, dom.Domain)
	}
	for _, dom := range configResp.Result.OtherCertConfig {
		domains = append(domains, dom.Domain)
	}

	log.Printf("got %d CDN domains to update", len(domains))
	err = batch(domains, 50, func(chunk []string) error {
		log.Printf("deploying %s", strings.Join(chunk, ", "))
		_, err = v.cCdn.BatchDeployCert(&cdn.BatchDeployCertRequest{
			CertId: certId,
			Domain: strings.Join(chunk, ","),
		})
		if err != nil {
			return fmt.Errorf("deploying cert: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("deploy cdn: %w", err)
	}

	log.Printf("cdn deploy finished")
	return nil
}

func (v *VolcDeployer) deployDcdn(certDomains []string, certId string) error {
	err, bindRes := v.listCertBind()
	if err != nil {
		return fmt.Errorf("dcdn list cert binds: %w", err)
	}

	domainIds := make([]string, 0)
	for _, bind := range bindRes.BindList {
		cdnDomains := strings.Split(bind.DomainName, ",")
		matched := matchDomain(certDomains, cdnDomains)
		log.Printf("checking dcdn domains: %s, matched: %v", cdnDomains, matched)
		if matched {
			domainIds = append(domainIds, bind.DomainId)
		}
	}

	log.Printf("got %d domains to deploy for dcdn", len(domainIds))
	if len(domainIds) > 0 {
		log.Printf("domain ids: %s", strings.Join(domainIds, ", "))
		err = v.createCertBind(certId, domainIds)
		if err != nil {
			return fmt.Errorf("dcdn create cert bind: %w", err)
		}
	}

	return nil
}

func (v *VolcDeployer) listCertBind() (error, *DcdnListCertBindResponse) {
	err, resp := volcRequest[DcdnListCertBindResponse](v, "ListCertBind", &DcdnListCertBindRequest{
		ProjectName: nil,
		SearchKey:   nil,
	})
	if err != nil {
		return fmt.Errorf("list cert bind dcdn: %w", err), nil
	}
	return nil, resp
}

func (v *VolcDeployer) createCertBind(certId string, domainIds []string) error {
	err, _ := volcRequest[interface{}](v, "CreateCertBind", &DcdnCreateCertBindRequest{
		CertId:     certId,
		DomainIds:  domainIds,
		CertSource: "volc",
	})
	if err != nil {
		return fmt.Errorf("create cert bind: %w", err)
	}
	return nil
}

func volcRequest[TResult any](v *VolcDeployer, api string, body interface{}) (error, *TResult) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err), nil
	}

	respBytes, _, err := v.cDcdn.Json(api, url.Values{}, string(bodyBytes))
	if err != nil {
		return fmt.Errorf("volcRequest %s: %w", api, err), nil
	}

	var resp ResponseBody[TResult]
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return fmt.Errorf("unmarshal response: %w", err), nil
	}

	if resp.ResponseMetadata.Error != nil {
		return fmt.Errorf("[%s] %s", resp.ResponseMetadata.Error.Code, resp.ResponseMetadata.Error.Message), nil
	}

	return nil, &resp.Result
}

func batch[TList any](list []TList, batchSize int, f func(chunk []TList) error) error {
	batches := (int)(math.Ceil(float64(len(list)) / float64(batchSize)))
	log.Printf("batches: %d", batches)

	for i := 0; i < batches; i++ {
		start := i * batchSize
		end := i*batchSize + batchSize
		if end > len(list) {
			end = len(list)
		}

		if end < start {
			break
		}

		chunk := list[start:end]
		err := f(chunk)
		if err != nil {
			return fmt.Errorf("batch %d: %w", i+1, err)
		}
	}
	return nil
}

type DcdnDescribeUserDomainsRequest struct {
	PageNum  int
	PageSize int
}

type DcdnDomainInfo struct {
	Domain string
	Status string
	Scope  string
}

type DcdnDescribeUserDomainsResponse struct {
	AllDomainNum    int
	OnlineDomainNum int
	Domains         []DcdnDomainInfo
	PageNum         int
	PageSize        int
}

type DcdnCreateCertBindRequest struct {
	CertId     string
	DomainIds  []string
	CertSource string
}

type DcdnListCertBindRequest struct {
	ProjectName *[]string
	SearchKey   *string
}

type DcdnListCertBindResponse struct {
	BindList []struct {
		CertId       string
		CertName     string
		CertSource   string
		DeployStatus string
		DomainName   string
		DomainId     string
		Expire       string
	}
}

type ResponseBody[TResult any] struct {
	ResponseMetadata struct {
		RequestId string
		Action    string
		Version   string
		Service   string
		Error     *struct {
			Code    string
			Message string
		}
	}
	Result TResult
}

var _ Deployer = (*VolcDeployer)(nil)

func matchDomain(certDomains []string, cdnDomains []string) bool {
	for _, cdnDom := range cdnDomains {
		matched := false
		cdnDom = strings.ToLower(cdnDom)
		for _, certDom := range certDomains {
			certDomLower := normalizeWildcardDomain(certDom)
			pos := strings.Index(cdnDom, certDomLower)
			if pos == -1 {
				continue
			}
			subPart := cdnDom[0:pos]
			if strings.Index(subPart, ".") == -1 {
				matched = true
			}
		}
		// all cert domains are not matching cdn domain
		if !matched {
			return false
		}
	}

	return true
}

func CreateVolcDeployer() (*VolcDeployer, error) {
	cCdn := cdn.NewInstance()
	cCdn.Client.SetAccessKey(os.Getenv("VOLC_ACCESS_KEY_ID"))
	cCdn.Client.SetSecretKey(os.Getenv("VOLC_SECRET_ACCESS_KEY"))

	cDcdn := volcBase.NewClient(&volcBase.ServiceInfo{
		Timeout: time.Minute * 5,
		Host:    "open.volcengineapi.com",
		Header: http.Header{
			"Accept":       []string{"application/json"},
			"Content-Type": []string{"application/json"},
		},
		Credentials: volcBase.Credentials{
			AccessKeyID:     os.Getenv("VOLC_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("VOLC_SECRET_ACCESS_KEY"),
			Service:         "dcdn",
			Region:          "cn-beijing",
		},
	}, map[string]*volcBase.ApiInfo{
		"DescribeUserDomains": {
			Method: "POST",
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"DescribeUserDomains"},
				"Version": []string{"2023-01-01"},
			},
		},
		"ListCertBind": {
			Method: "POST",
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"ListCertBind"},
				"Version": []string{"2021-04-01"},
			},
		},
		"CreateCertBind": {
			Method: "POST",
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"CreateCertBind"},
				"Version": []string{"2021-04-01"},
			},
		},
	})

	return &VolcDeployer{cCdn: cCdn, cDcdn: cDcdn}, nil
}
