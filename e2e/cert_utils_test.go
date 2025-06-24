/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package e2e

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	"math/big"
	"net"
	"os"
	"time"
)

// generateSelfSignedCert creates a self-signed certificate and returns the cert, key, and a cleanup function
func generateSelfSignedCert() (func(), error) {
	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create a certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // Valid for 1 year

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Steadybit Test"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "host.minikube.internal"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	// Create the certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Create temporary certificate file
	certFile, err := os.CreateTemp("", "cert*.pem")
	if err != nil {
		return nil, err
	}
	defer certFile.Close()

	// Write certificate to file
	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return nil, err
	}

	// Create temporary key file
	keyFile, err := os.CreateTemp("", "key*.pem")
	if err != nil {
		return nil, err
	}
	defer keyFile.Close()

	// Write private key to file
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	err = pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
	if err != nil {
		return nil, err
	}

	// Set environment variables for later use
	err = os.Setenv("CERT_FILE", certFile.Name())
	if err != nil {
		return nil, err
	}
	err = os.Setenv("KEY_FILE", keyFile.Name())
	if err != nil {
		return nil, err
	}

	cleanup := func() {
		// delete the temporary files
		if err := os.Remove(certFile.Name()); err != nil {
			log.Error().Err(err).Msgf("Failed to remove temporary certificate file: %s", certFile.Name())
		}
		if err := os.Remove(keyFile.Name()); err != nil {
			log.Error().Err(err).Msgf("Failed to remove temporary key file: %s", keyFile.Name())
		}
		// Unset environment variables
		if err := os.Unsetenv("CERT_FILE"); err != nil {
			log.Error().Err(err).Msg("Failed to unset CERT_FILE environment variable")
		}
		if err := os.Unsetenv("KEY_FILE"); err != nil {
			log.Error().Err(err).Msg("Failed to unset KEY_FILE environment variable")
		}
	}

	return cleanup, nil
}
// installConfigMap creates a ConfigMap with the self-signed certificate in the minikube cluster
func installConfigMap(m *e2e.Minikube) error {
	err := m.CreateConfigMap("default", "gatling-self-signed-ca", os.Getenv("CERT_FILE"))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create ConfigMap with self-signed certificate")
		return err
	}
	return nil
}
