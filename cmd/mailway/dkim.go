package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"

	"github.com/pkg/errors"

	mconfig "github.com/mailway-app/config"
	log "github.com/sirupsen/logrus"
)

func generateDKIM() ([]byte, error) {
	config, err := mconfig.Read()
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not read config")
	}

	certPath := fmt.Sprintf("/etc/ssl/certs/dkim-%s.pem", config.InstanceHostname)
	privPath := fmt.Sprintf("/etc/ssl/private/dkim-%s.pem", config.InstanceHostname)

	if fileExists(certPath) || fileExists(privPath) {
		log.Warnf("%s or %s already exist; skipping DKIM key generation.", certPath, privPath)
		return []byte{}, nil
	}

	reader := rand.Reader
	bitSize := 2048

	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not generate RSA key")
	}

	// certificate
	if err := saveRSACert(certPath, &key.PublicKey); err != nil {
		return []byte{}, errors.Wrap(err, "could not save certificate")
	}

	// private key
	if err := savePrivateKey(privPath, key); err != nil {
		return []byte{}, errors.Wrap(err, "could not save private key")
	}

	err = mconfig.WriteDKIM(privPath)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not write DKIM config")
	}

	// encode key for DNS
	bytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not marshal public key")
	}
	return bytes, nil
}
