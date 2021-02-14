package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"time"

	mconfig "github.com/mailway-app/config"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	httpClient = http.Client{
		Timeout: time.Second * 5,
	}
)

// instance's config sent by the Mailay API
type instanceConfig struct {
	Hostname string `json:"hostname"`
	Email    string `json:"email"`
}

type JWTClaims struct {
	jwt.StandardClaims
	Data instanceConfig `json:"data"`
}

type authorizeRes struct {
	Ok   bool `json:"ok"`
	Data struct {
		JWT string `json:"jwt"`
	} `json:"data"`
}

func lookupPublicKey() (interface{}, error) {
	data, err := ioutil.ReadFile(path.Join(mconfig.ROOT_LOCATION, "key.pub"))
	if err != nil {
		return nil, errors.Wrap(err, "could not read key file")
	}
	return jwt.ParseRSAPublicKeyFromPEM(data)
}

func authorize(serverId string) (string, error) {
	url := fmt.Sprintf("%s/instance/%s/authorize", API_BASE_URL, serverId)
	log.Debugf("request to %s", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "mailway-self-host")

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		return "", errors.Wrap(getErr, "could not send request")
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return "", errors.Wrap(readErr, "couldn't read body")
	}

	d := authorizeRes{}
	jsonErr := json.Unmarshal(body, &d)
	if jsonErr != nil {
		return "", errors.Wrap(jsonErr, "could not parse JSON")
	}

	err = mconfig.WriteServerJWT(d.Data.JWT)
	if err != nil {
		return "", errors.Wrap(jsonErr, "could not write JWT")
	}

	return d.Data.JWT, nil
}

func parseJWT(v string) (*jwt.Token, error) {
	claims := new(JWTClaims)
	token, err := jwt.ParseWithClaims(v, claims, func(token *jwt.Token) (interface{}, error) {
		return lookupPublicKey()
	})

	if err != nil || !token.Valid {
		return nil, errors.Wrap(err, "key failed to verify")
	}
	if err := token.Claims.Valid(); err != nil {
		return nil, errors.Wrap(err, "JWT claims not valid")
	}

	return token, nil
}

func getJWTData(token *jwt.Token) (instanceConfig, error) {
	claims := token.Claims.(*JWTClaims)
	return claims.Data, nil
}
