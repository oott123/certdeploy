package deployer

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestUpyunDeployer_Login(t *testing.T) {
	u, err := CreateUpyunDeployer()
	if err != nil {
		panic(err)
	}

	err = u.Login()
	if err != nil {
		panic(err)
	}

	id, err := u.UploadCertificate(readFile(os.Getenv("CERT_PATH")), readFile(os.Getenv("CERT_KEY_PATH")))

	if err != nil {
		panic(err)
	}

	domains, err := u.DomainsByCertificate(id)
	if err != nil {
		panic(err)
	}

	fmt.Println("domains", strings.Join(domains, ","))

	err = u.SetDomainCertificate(id, domains[0])
	if err != nil {
		panic(err)
	}
}

func readFile(filename string) string {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	return string(bytes)
}
