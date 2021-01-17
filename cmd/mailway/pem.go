package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func saveRSACert(name string, pubkey *rsa.PublicKey) error {
	log.Debugf("write certificate %s", name)
	f, err := os.Create(name)
	if err != nil {
		return errors.Wrap(err, "could not create file")
	}
	bytes := x509.MarshalPKCS1PublicKey(pubkey)
	err = pem.Encode(f, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: bytes,
	})
	if err != nil {
		return errors.Wrap(err, "could not encode cert")
	}
	return nil
}

func saveCert(name string, cert *x509.Certificate) error {
	log.Debugf("write certificate %s", name)
	f, err := os.Create(name)
	if err != nil {
		return errors.Wrap(err, "could not create file")
	}
	err = pem.Encode(f, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
	if err != nil {
		return errors.Wrap(err, "could not encode cert")
	}
	return nil
}

func savePrivateKey(name string, key crypto.PrivateKey) error {
	log.Debugf("write private key %s", name)
	f, err := os.Create(name)
	if err != nil {
		return errors.Wrap(err, "could not create file")
	}
	err = pem.Encode(f, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key.(*rsa.PrivateKey)),
	})
	if err != nil {
		return errors.Wrap(err, "could not encode private key")
	}

	return nil
}
