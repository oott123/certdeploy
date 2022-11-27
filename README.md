# certdeploy

Deploy https certificates non-interactively to CDN services.

## Supported deployers

### CDN Providers

* Aliyun
* Upyun
* Tencent Cloud

Deploys to all CDN domains which matched by given certificate.

### Azure KeyVault

Updates all certificates in specified KeyVault, if only all domains in existing 
certificate are covered by given certificate.

## Environment Variables

* `CERT_PATH` - Certificate file path, should contain certificate and all intermediate certificates. `LEGO_CERT_PATH` is also supported.
* `CERT_KEY_PATH` - Certificate key file path, should contain private key for certificate. `LEGO_CERT_KEY_PATH` is also supported.
* `CERT_DEPLOYER` - Deployer vendor. Default: `aliyun`

### Aliyun deployer

* `CERT_DEPLOYER` - `aliyun`
* `ALIYUN_ACCESS_KEY_ID` - Access key ID for aliyun CDN. User should have `AliyunCDNFullAccess` permission.
* `ALIYUN_ACCESS_KEY_SECRET` - Access key secret for aliyun CDN.
* `ALIYUN_CERT_UPDATE_ONLY` - If `true`, only certs for CDN domains with SSL enabled will be updated. Default: `false`
* `ALIYUN_CERT_RESOURCE_GROUP` - If given, only certs for domains under this resource group will be updated. Default: `(empty)`

### Upyun deployer

* `CERT_DEPLOYER` - `upyun`
* `UPYUN_USERNAME` - Upyun login username
* `UPYUN_PASSWORD` - Upyun login password. 2FA is not supported now.

### tencentcloud deployer

* `CERT_DEPLOYER` - `tencentcloud`
* `TENCENTCLOUD_SECRET_ID` - Secret ID for tencent cloud.
* `TENCENTCLOUD_SECRET_KEY` - Secret Key for tencent cloud.
* `TENCENTCLOUD_CERT_UPDATE_ONLY` - If `true`, only certs for CDN domains with SSL enabled will be updated. Default: `false`

### udomain deployer

* `CERT_DEPLOYER` - `udomain`
* `UDOMAIN_API_KEY` - API Key created from [udomain CDN dashboard](https://cdn.8338.hk/key)

### azure deployer

* `CERT_DEPLOYER` - `azure`
* `AZURE_KEY_VAULT_URI` - Azure KeyVault Uri, likely `https://SOMETHING.vault.azure.net/`
* Follow [Azure authentication with the Azure SDK for Go](https://learn.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication) 
  and [Assign a Key Vault access policy](https://learn.microsoft.com/en-us/azure/key-vault/general/assign-access-policy)
  to configure credentials
