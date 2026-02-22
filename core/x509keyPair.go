package core

import (
	mrand "math/rand"

	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

// certSeed 固定种子使每次生成的密钥/证书字节完全相同，重启后无需重新信任
const certSeed = 623532

var certPEM, keyPEM = func() ([]byte, []byte) {
	r := mrand.New(mrand.NewSource(certSeed))
	key, _ := rsa.GenerateKey(r, 2048)
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(certSeed),
		Subject: pkix.Name{
			Province:           []string{"Lotus Land Story"},
			CommonName:         "ShangHai",
			Organization:       []string{"Arisu"},
			OrganizationalUnit: []string{"Gengakudan"},
		},
		NotBefore: time.Date(1996, 11, 3, 0, 0, 0, 0, time.UTC),
		NotAfter:  time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	der, _ := x509.CreateCertificate(r, &tmpl, &tmpl, &key.PublicKey, key)
	c := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	kDer, _ := x509.MarshalPKCS8PrivateKey(key)
	k := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: kDer})

	return c, k
}()
