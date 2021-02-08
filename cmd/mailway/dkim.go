package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	"github.com/pkg/errors"

	mconfig "github.com/mailway-app/config"
	log "github.com/sirupsen/logrus"
)

func getDNSKey(pubKeyPath string) ([]byte, error) {
	bytes, err := ioutil.ReadFile(pubKeyPath)
	if err != nil {
		return []byte{}, errors.New("could not read public key file")
	}

	pubPem, _ := pem.Decode(bytes)
	if pubPem == nil {
		return []byte{}, errors.New("public key is not in PEM format")
	}

	pubKey, err := x509.ParsePKCS1PublicKey(pubPem.Bytes)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not read public RSA key")
	}

	// encode key for DNS
	bytes, err = x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not marshal public key")
	}
	return bytes, nil
}

func generateDKIM() ([]byte, error) {
	certPath := "/etc/ssl/certs/mailway-dkim.pem"
	privPath := CONFIG.OutDKIMPath

	if fileExists(certPath) || fileExists(privPath) {
		log.Warnf("%s or %s already exist; skipping DKIM key generation.", certPath, privPath)
		return getDNSKey(certPath)
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

	return getDNSKey(certPath)
}
