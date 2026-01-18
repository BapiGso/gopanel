package core

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

var certPEM, keyPEM = func() ([]byte, []byte) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(623532),
		Subject: pkix.Name{
			Province:           []string{"Lotus Land Story"},
			CommonName:         "ShangHai",
			Organization:       []string{"Arisu"},
			OrganizationalUnit: []string{"Gengakudan"}},
		NotBefore: time.Date(1996, 11, 3, 0, 0, 0, 0, time.UTC),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	// 直接生成 DER 并编码为 PEM
	der, _ := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	c := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	// 缩减私钥编码步骤
	kDer, _ := x509.MarshalPKCS8PrivateKey(key)
	k := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: kDer})

	return c, k
}()
