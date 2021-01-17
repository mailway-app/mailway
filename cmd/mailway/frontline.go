package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"text/template"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme/autocert"
)

func generateFrontlineConf() error {
	file := "/etc/mailway/frontline/nginx.conf"
	if fileExists(file) {
		log.Warnf("%s already exists; skipping frontline config generation.", file)
		return nil
	}

	tmpl := template.Must(
		template.ParseFiles("/etc/mailway/frontline/nginx.conf.tmpl"))

	dest, err := os.Create(file)
	if err != nil {
		return errors.Wrap(err, "could not create conf file")
	}
	defer dest.Close()

	err = tmpl.Execute(dest, CONFIG)
	if err != nil {
		return errors.Wrap(err, "failed to render template")
	}

	return nil
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("HTTP %s %s\n", r.Method, r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func generateHTTPCert() error {
	certPath := fmt.Sprintf("/etc/ssl/certs/http-%s.pem", CONFIG.InstanceHostname)
	privPath := fmt.Sprintf("/etc/ssl/private/http-%s.pem", CONFIG.InstanceHostname)
	if fileExists(certPath) || fileExists(privPath) {
		log.Warnf("%s or %s already exist; skipping HTTPS key generation.", certPath, privPath)
		return nil
	}

	m := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Email:  CONFIG.InstanceEmail,
	}

	h := loggingMiddleware(m.HTTPHandler(nil))
	go func() {
		log.Fatal(http.ListenAndServe(":http", h))
	}()

	log.Infof("asking a certificate for %s; this could take a minute or two", CONFIG.InstanceHostname)
	cert, err := m.GetCertificate(&tls.ClientHelloInfo{
		ServerName: CONFIG.InstanceHostname,
	})
	log.Info("OK")
	if err != nil {
		return errors.Wrap(err, "could not generate certificate")
	}

	// certificate
	if err := saveCert(certPath, cert.Leaf); err != nil {
		return errors.Wrap(err, "could not save certificate")
	}

	// private key
	if err := savePrivateKey(privPath, cert.PrivateKey); err != nil {
		return errors.Wrap(err, "could not save private key")
	}

	return nil
}
