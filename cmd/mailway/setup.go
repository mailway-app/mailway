package main

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"github.com/mailway-app/config"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func setup() error {
	if err := runPreflightChecks(); err != nil {
		return errors.Wrap(err, "preflight checks failed")
	}
	log.Info("preflight check passed")

	dkim, err := generateDKIM()
	if err != nil {
		return errors.Wrap(err, "could not generate DKIM keys")
	}

	ip, err := GetOutboundIP()
	if err != nil {
		return errors.Wrap(err, "could not get outbound IP")
	}
	url := fmt.Sprintf(
		"https://dash.mailway.app/helo?server_id=%s&ip=%s&dkim=%s",
		config.CurrConfig.ServerId, ip, url.QueryEscape(base64.StdEncoding.EncodeToString(dkim)))
	fmt.Printf("Open %s\n", url)

	ticker := time.NewTicker(2 * time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			jwt, err := authorize(config.CurrConfig.ServerId)
			if err != nil {
				panic(err)
			}
			if jwt == "" {
				continue
			}
			ticker.Stop()
			log.Info("instance connected with Mailway")
			token, err := parseJWT(jwt)
			if err != nil {
				panic(err)
			}
			data, err := getJWTData(token)
			if err != nil {
				panic(err)
			}
			err = config.WriteInstanceConfig(data.Hostname, data.Email)
			if err != nil {
				panic(err)
			}

			if err := generateFrontlineConf(); err != nil {
				return errors.Wrap(err, "could not generate frontline conf")
			}
			if err := generateHTTPCert(); err != nil {
				return errors.Wrap(err, "could not generate certificates for HTTP")
			}

			log.Info("Setup completed; starting email service")
			services("start")
			close(quit)
		case <-quit:
			ticker.Stop()
			return nil
		}
	}
}
