package main

import (
	"fmt"
	"github.com/oott123/certdeploy/pkg/certparser"
	"github.com/oott123/certdeploy/pkg/deployer"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	certFile := getEnv("CERT_PATH", "LEGO_CERT_PATH")
	keyFile := getEnv("CERT_KEY_PATH", "LEGO_CERT_KEY_PATH")
	deployerName := getEnv("CERT_DEPLOYER")
	if deployerName == "" {
		deployerName = "aliyun"
	}

	if certFile == "" || keyFile == "" {
		fmt.Println("no cert file and/or key file given")
		os.Exit(1)
		return
	}

	dp, err := deployer.Create(deployerName)
	if err != nil {
		log.Fatalf("failed to create deployer: %s", err)
	}

	log.Printf("deploying cert %s, key %s using deployer: %s", certFile, keyFile, dp.Name())

	cert, err := ioutil.ReadFile(certFile)
	if err != nil {
		log.Fatalf("failed to read cert file %s: %s", certFile, err)
	}
	key, err := ioutil.ReadFile(keyFile)
	if err != nil {
		log.Fatalf("failed to read key file %s: %s", keyFile, err)
	}

	domains, err := certparser.DomainsFromCert(string(cert))
	if err != nil {
		log.Fatalf("failed to parse domains from cert: %s", err)
	}

	err = dp.Deploy(domains, string(cert), string(key))
	if err != nil {
		log.Fatalf("failed to deploy: %s", err)
	}

	log.Println("finished deploy cert")
}

func getEnv(keys ...string) string {
	for _, key := range keys {
		value := os.Getenv(key)
		if value != "" {
			return value
		}
	}
	return ""
}
