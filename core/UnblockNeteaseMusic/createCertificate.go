package UnblockNeteaseMusic

import (
	"io/fs"
	"os"
)

func createCertificate() error {
	_, errCert := os.Stat("server.crt")
	_, errKey := os.Stat("server.key")
	if os.IsNotExist(errCert) {
		err := os.WriteFile("server.crt", []byte(certContent), fs.ModePerm)
		if err != nil {
			return err
		}
	}
	// 如果server.key不存在，写入私钥内容
	if os.IsNotExist(errKey) {
		err := os.WriteFile("server.key", []byte(keyContent), fs.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

const certContent = `
-----BEGIN CERTIFICATE-----
MIIDtDCCApygAwIBAgIUakEZumrWTMH3PZcrR0LkjtUa/jYwDQYJKoZIhvcNAQEL
BQAwUTELMAkGA1UEBhMCQ04xJDAiBgNVBAMMG1VuYmxvY2tOZXRlYXNlTXVzaWMg
Um9vdCBDQTEcMBoGA1UECgwTVW5ibG9ja05ldGVhc2VNdXNpYzAeFw0yMTExMDQx
MzA5NTFaFw0yNDAyMDcxMzA5NTFaMHsxCzAJBgNVBAYTAkNOMREwDwYDVQQHDAhI
YW5nemhvdTEsMCoGA1UECgwjTmV0RWFzZSAoSGFuZ3pob3UpIE5ldHdvcmsgQ28u
LCBMdGQxETAPBgNVBAsMCElUIERlcHQuMRgwFgYDVQQDDA8qLm11c2ljLjE2My5j
b20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCzgnpbiz0qHYaCjK95
/3+B5egJrttJ80gR2zEA7rL34gd3b4rMK0NtJyuawzeF58klSh15xNxV5hvCXZ/d
d6ww/dDSqPU8EsYHrqpS0cKY4DWH4deucge9+oFT+RIcFJMWlxphLxfXVvjdZhb7
CcPmtiBWFwIK6Sj+ggPUk/Ntq4mA9xBn13LH0mlPEFkkI7Qxnapp4gMxdPnshM01
GbMKmDyxh2o3pK81fsp7W+mV06uDALFKF65D3G1m4lYxRIlJBmSmpcCaOMK29kEV
A8eHj0E/xg7nAtyuK6eoudDOXdELXPHL6G2izob3HagGkphNcp/IR7FGQ3pysDTI
knXdAgMBAAGjWjBYMAkGA1UdEwQCMAAwCwYDVR0PBAQDAgTwMBMGA1UdJQQMMAoG
CCsGAQUFBwMBMCkGA1UdEQQiMCCCDW11c2ljLjE2My5jb22CDyoubXVzaWMuMTYz
LmNvbTANBgkqhkiG9w0BAQsFAAOCAQEAHv+kmvRWx5vBrLN4BQLzqXaeDKqVs1+P
QCTi2D+/nWk0eudQJnIF2UqlmL815k7fGHVi0aAAjKBv1iA566ln4oV6hfgjnjW6
NL/ETkwOLTwtfJvHTWqYSdFbLqDVSLiLgGX0hqD/lpGSzwk2Z/LOjAJo/w+JmwXR
5dDk/9vwTzNl/Om3uHhvMHu7rIss737cZg4XUq0aGYQt+kTH15+m4wMvkGl4xxrE
rSac48e0QiGLFDZuCmaHVxpUAAmREnap33RmG+nHpwFP9G0lpurIc6TaH6ScpKSu
wRUvnWV6pP3+eUZ4Ap2/zwguYu4mNgr7vXA0l/yJgWBBV+qWiSfrTQ==
-----END CERTIFICATE-----
`

const keyContent = `
-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAs4J6W4s9Kh2Ggoyvef9/geXoCa7bSfNIEdsxAO6y9+IHd2+K
zCtDbScrmsM3hefJJUodecTcVeYbwl2f3XesMP3Q0qj1PBLGB66qUtHCmOA1h+HX
rnIHvfqBU/kSHBSTFpcaYS8X11b43WYW+wnD5rYgVhcCCuko/oID1JPzbauJgPcQ
Z9dyx9JpTxBZJCO0MZ2qaeIDMXT57ITNNRmzCpg8sYdqN6SvNX7Ke1vpldOrgwCx
SheuQ9xtZuJWMUSJSQZkpqXAmjjCtvZBFQPHh49BP8YO5wLcriunqLnQzl3RC1zx
y+htos6G9x2oBpKYTXKfyEexRkN6crA0yJJ13QIDAQABAoIBAAImcPLBwzTK776G
kt+COPPEXjgneQb0vAtCtd6N/WTMt0wt8NqrNf6LtpD+/55B/X3N2naH7h+1RuXY
Gz8a3NwlXif30CAtFWQoKnAdhRgxr1J2WRAE26Th6ESqOhZOBMkDfFRnrQBuUULN
oz7Uih0sV0zQr7yTuGL8CbG1J/hLTY/kM6Q6NOlWgNvhq0oJMgIoKcDdTfjmP8sq
Pd2GNG5nnUMZfZFeHBalhQWj/WrDdN6Ctvp9YgVEgUzDXLb9IAtS5mWZPqpsSfGs
U1T3nuSrLyatvMsBGR3BfHaBzOOggSA+02NLBRVQMIdu0sNmr9YQr2MoarwfZX2e
yzl5SEECgYEA5LXq6OVzhs7l1yYDsX4gbeUElZRGxb0BYS3Bx1Ha6kIFNRbYVdfI
ynmyoyXUwXD9D2YxuS8xwcylfnrAZaYxBUT8t54U8Eyg2PPIVoJwDLdq/X5Qj/zZ
ATojuVQaGoRHDBhOaBtMdIcxHSBIN5fcCHeNK0sWtegqNz9EU8ohMW0CgYEAyO2x
Uhcl6ioIOfvfmGrT+TimJ710fFQYDv1dQ+0BfUCR1XnTtn/sYGkXmB4Vpm5BXnzS
ZZzmaNRP2+uXL15zYfQTjOlgIusfIYpY9jebRp8y/6R4mmR99JyZToleXCR1GNet
zSgpDtlTHFjWrf61gZE+ArnpmQf2KxAbtI4YADECgYAtlzXkhxioXsXiRWmnEAVW
4rgvOQeCk1KbFIv0N5Tz7YUsOAmX0xPriKbbbsciaGuJjk2LJKU+hJTYyY9gs+hB
rKbT02dJH31QwgfFdurvHgDt1ygoC7cWT4ifgOxTLNscxhubFRYAhJJ9w9mhe1wZ
M/uoDafRSx5fNPVh3oEPYQKBgD0q1xNlhp5No2l7euscgmFZVIO+kiqTMyvFB9J4
4T4vHPY++yuQr/X9qDgf2HltESthlm9mn2IEWDdb9g9uknOcaSM5nJGkwDpmsoRq
EzQhnNXkTO67bvi7f5RAD2C/nIIujnNpKW6izEFR7jDT3I+QFq+fxzRWDyO26KhD
pZzRAoGAVUVLToN/+JRaSAksRu2+iAhm0k8bJpqKv6CwHqeu92HsPNjyed67yWfl
Tj8tNj/P7J++rmLfefmH3flsJp1xtwuCjl926AEGXA2q2l8wJJbUwEDVmWB6z7tj
DfoWAtSTNkckozFriPN93MRAGM8tHeXXtWubhPSCHmIu8J7jBdY=
-----END RSA PRIVATE KEY-----
`
