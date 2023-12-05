package deployer

import "fmt"

type Deployer interface {
	Name() string
	Deploy(domains []string, cert, key string) error
}

func Create(name string) (Deployer, error) {
	if name == "aliyun" {
		return CreateAliyunDeployer()
	} else if name == "upyun" {
		return CreateUpyunDeployer()
	} else if name == "tencentcloud" {
		return CreateTencentCloudDeployer()
	} else if name == "udomain" {
		return CreateUDomainDeployer()
	} else if name == "azure" {
		return CreateAzureDeployer()
	} else if name == "volc" {
		return CreateVolcDeployer()
	} else {
		return nil, fmt.Errorf("create deployer failed: no deployer named %s", name)
	}
}
