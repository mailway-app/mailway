package main

import (
	"bytes"
	"io/ioutil"
	"net/mail"
	"os"
	"path"
	"strings"
	"time"

	"github.com/mailway-app/config"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	JWT_CHECK_INTERVAL = 1 * time.Hour

	MAILOUT_RETRY_INTERVAL = 3 * time.Minute
	MAILWAY_RETRY_SEQ      = []int{-1, 0, 1, 1, 2, 3, 5, 8, 13, 21, 34, 55, 89, 144,
		233, 377, 610, 987, 1597, 2584, 4181}
)

func supervise() error {
	done := make(chan interface{})

	if !config.CurrConfig.IsInstanceLocal() {
		go func() {
			if err := superviseServerJWT(); err != nil {
				log.Fatalf("failed to supervise server JWT: %s", err)
			}
		}()
	}
	go func() {
		if err := superviseMailoutRetrier(); err != nil {
			log.Fatalf("failed to supervise mailout retrier: %s", err)
		}
	}()

	<-done
	return nil
}

func getNextRetry(modTime time.Time, retry int) (time.Time, error) {
	if retry > len(MAILWAY_RETRY_SEQ) {
		return time.Time{}, errors.Errorf("too many retries (%d), ignoring", retry)
	}
	n := time.Duration(MAILWAY_RETRY_SEQ[retry]) * 3 * time.Minute
	return modTime.Add(n), nil
}

func retryCount(data []byte) int {
	return strings.Count(string(data), "Mw-Int-Id")
}

func superviseMailoutRetrier() error {
	log.Info("mailout retrier running")
	for range time.Tick(MAILOUT_RETRY_INTERVAL) {
		files, err := ioutil.ReadDir(config.RUNTIME_LOCATION)
		if err != nil {
			return errors.Wrapf(err, "failed to read %s", config.RUNTIME_LOCATION)
		}

		for _, file := range files {
			abspath := path.Join(config.RUNTIME_LOCATION, file.Name())
			data, err := ioutil.ReadFile(abspath)
			if err != nil {
				return errors.Wrap(err, "could not read file")
			}

			// ensures that the file has been in the queue for long enough
			if time.Since(file.ModTime()) < MAILOUT_RETRY_INTERVAL {
				continue
			}

			retryCount := retryCount(data)
			nextRetry, err := getNextRetry(file.ModTime(), retryCount)
			if err != nil {
				log.Errorf("could not retry: %s", err)
				continue
			}

			msg, err := mail.ReadMessage(bytes.NewReader(data))
			if err != nil {
				return errors.Wrap(err, "could not read message")
			}

			via := msg.Header.Get("Mw-Int-Via")
			if via == "responder" {
				log.Warnf("retry not yet supported for responder")
				continue
			}

			if nextRetry.Before(time.Now()) {
				log.Infof("%s retried %d time(s), retyring now", abspath, retryCount)
				if err := recoverEmail(abspath); err != nil {
					log.Errorf("failed to recover email: %s", err)

					// delete the email since a new buffer will be created from
					// the failure
					if err := os.Remove(abspath); err != nil {
						return errors.Wrap(err, "could not delete file")
					}
				}
			} else {
				log.Infof("%s retried %d time(s) next retry in %v",
					abspath, retryCount, time.Until(nextRetry))
			}
		}
	}
	return nil
}

func superviseServerJWT() error {
	log.Info("server JWT watcher running")
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
