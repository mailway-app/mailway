package main

import (
	"time"

	"github.com/mailway-app/config"

	log "github.com/sirupsen/logrus"
)

var (
	JWT_CHECK_INTERVAL = 1 * time.Hour
)

func supervise() error {
	done := make(chan interface{})
	go func() {
		if err := superviseServerJWT(); err != nil {
			log.Fatalf("failed to supervise server JWT: %s", err)
		}
	}()

	log.Info("supervisor running")
	<-done
	return nil
}

func superviseServerJWT() error {
	for range time.Tick(JWT_CHECK_INTERVAL) {
		token := config.CurrConfig.ServerJWT
		if token == "" {
			log.Warn("no existing token; did you forgot to run mailway setup?")
			continue
		}

		jwt, err := parseJWT(token)
		if err != nil {
			log.Warnf("failed to parse exiting token: %s; asking new one", err)
			if err := newJWT(); err != nil {
				log.Errorf("failed to get new JWT: %s", err)
			}
			continue
		}

		claims := jwt.Claims.(*JWTClaims)

		now := time.Now().Unix()
		delta := time.Unix(claims.ExpiresAt, 0).Sub(time.Unix(now, 0))

		log.Debugf("server JWT has %v to live", delta)

		if delta < 24*time.Hour {
			log.Infof("Token almost expired; asking new one")
			if err := newJWT(); err != nil {
				log.Errorf("failed to get new JWT: %s", err)
			}
		}

	}

	return nil
}
