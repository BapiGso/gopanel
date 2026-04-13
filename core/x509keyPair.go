package core

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

const (
	certFilePath = "server.crt"
	keyFilePath  = "server.key"
)

func loadOrCreateCertificate() ([]byte, []byte, error) {
	if certPEM, keyPEM, err := loadCertificateFromDisk(); err == nil {
		return certPEM, keyPEM, nil
	}

	certPEM, keyPEM, err := generateCertificatePair()
	if err != nil {
		return nil, nil, fmt.Errorf("generate tls certificate: %w", err)
	}
	if err := persistCertificatePair(certPEM, keyPEM); err != nil {
		return nil, nil, fmt.Errorf("persist tls certificate: %w", err)
	}
	return certPEM, keyPEM, nil
}

func loadCertificateFromDisk() ([]byte, []byte, error) {
	certPEM, err := os.ReadFile(certFilePath)
	if err != nil {
		return nil, nil, err
	}
	keyPEM, err := os.ReadFile(keyFilePath)
	if err != nil {
		return nil, nil, err
	}

	if _, err := tlsX509KeyPair(certPEM, keyPEM); err != nil {
		return nil, nil, err
	}
	return certPEM, keyPEM, nil
}

func generateCertificatePair() ([]byte, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	tmpl := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "gopanel",
			Organization: []string{"gopanel"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, nil
}

func persistCertificatePair(certPEM, keyPEM []byte) error {
	if err := os.WriteFile(certFilePath, certPEM, 0600); err != nil {
		return err
	}
	if err := os.WriteFile(keyFilePath, keyPEM, 0600); err != nil {
		return err
	}
	return nil
}

func tlsX509KeyPair(certPEM, keyPEM []byte) (tls.Certificate, error) {
	return tls.X509KeyPair(certPEM, keyPEM)
}
