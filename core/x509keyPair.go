package core

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"time"
)

var certPEM, keyPEM = func() ([]byte, []byte) {
	// 1. 生成私钥（2048位 RSA）
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("无法生成私钥: %v", err)
	}
	serialNumber := big.NewInt(0)
	serialNumber.SetBytes([]byte{0x46, 0x78, 0x58, 0xB7})
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:            []string{"JP"},
			Province:           []string{"Lotus Land Story"},
			Locality:           []string{"ShangHai"},
			Organization:       []string{"Arisu"},
			OrganizationalUnit: []string{"Gengakudan"},
			CommonName:         "ShangHai",
		},
		NotBefore:             time.Date(2024, 5, 10, 8, 38, 32, 0, time.UTC),
		NotAfter:              time.Date(2034, 5, 8, 8, 38, 32, 0, time.UTC),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true, // 这是一个 CA 证书
	}
	// 3. 创建自签名证书
	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template, // 自签名：签发者是自己
		&privateKey.PublicKey,
		privateKey, // 用自己的私钥签名
	)
	if err != nil {
		log.Fatalf("无法创建证书: %v", err)
	}
	// 4. 编码证书为 PEM 格式
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})
	// 5. 编码私钥为 PEM 格式（PKCS#8）
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Fatalf("无法编码私钥: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDER,
	})
	return certPEM, keyPEM
}()
