# certdeploy

Deploy https certificates non-interactively to CDN services.

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
