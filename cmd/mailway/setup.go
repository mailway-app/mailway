package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/mailway-app/config"

	valid "github.com/asaskevich/govalidator"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func prompConfirm(isOptional bool, msg string) {
	label := fmt.Sprintf("Did you %s", msg)
	if v := os.Getenv("DEBIAN_FRONTEND"); v == "noninteractive" {
		log.Infof("%s? Assuming yes because output is not a tty", label)
		return
	}
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
		Default:   "y",
	}

	_, err := prompt.Run()
	if err != nil {
		if !isOptional {
			log.Fatalf("Before proceeding with the setup you need to: %s", msg)
		}
	}
}

func getText(msg string, validation string) string {
	validate := func(input string) error {
		switch validation {
		case "domain":
			if !valid.IsDNSName(input) {
				return errors.New("Domain name must be valid")
			}
		case "email":
			if !valid.IsEmail(input) {
				return errors.New("Email address must be valid")
			}
		default:
			panic("unknown validation")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    msg,
		Validate: validate,
	}
	result, err := prompt.Run()
	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
		return ""
	}
	return result
}

func setupConnected(ip *net.IP, dkim string) error {
	url := fmt.Sprintf(
		"https://dash.mailway.app/helo?server_id=%s&ip=%s&dkim=%s",
		config.CurrConfig.ServerId, ip, url.QueryEscape(dkim))
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
			err = config.WriteInstanceConfig("connected", data.Hostname, data.Email)
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

func setupLocal(ip *net.IP, dkim string) error {
	var hostname string
	var email string

	if v := os.Getenv("MW_HOSTNAME"); v != "" {
		hostname = v
	} else {
		hostname = getText("Please enter the name of your email server (for example: mx.example.com)", "domain")
	}

	dnsFields := func(name, value string) string {
		return fmt.Sprintf("Name: %s\nValue:\n\n%s\n", name, value)
	}

	fmt.Printf("Add a DNS record (type A);\n%s\n", dnsFields(hostname, ip.String()))
	prompConfirm(false, "add the A DNS record")

	fmt.Printf("Optionally, add a DNS record (type TXT):\n%s\n",
		dnsFields(hostname, fmt.Sprintf("v=spf1 ip4:%s/32 ~all", ip)))
	prompConfirm(true, "add the TXT DNS record")

	fmt.Printf("Optionally, add a DNS record (type TXT):\n%s\n",
		dnsFields("smtp._domainkey."+hostname, fmt.Sprintf("v=DKIM1; k=rsa; p=%s", dkim)))
	prompConfirm(true, "add the TXT DNS record")

	if v := os.Getenv("MW_EMAIL"); v != "" {
		email = v
	} else {
		email = getText("Please enter your email address (email will only be used to generate certificates)", "email")
	}

	err := config.WriteInstanceConfig("local", hostname, email)
	if err != nil {
		return errors.Wrap(err, "could not write instance config")
	}

	if err := generateFrontlineConf(); err != nil {
		return errors.Wrap(err, "could not generate frontline conf")
	}

	log.Info("Setup completed; starting email service")
	services("start")
	return nil
}

func setup() error {
	if isLocalSetup {
		log.Info("Setup for local mode")
	}

	if err := runPreflightChecks(); err != nil {
		if v := os.Getenv("DEBIAN_FRONTEND"); v == "noninteractive" {
			log.Infof("The preflight checks failed because: %s. Ignoring and continuing because output is not a tty", err)
		} else {
			prompt := promptui.Prompt{
				Label:     fmt.Sprintf("The preflight checks failed because: %s. Confirm to ignore and continue the setup", err),
				IsConfirm: true,
			}

			if _, err := prompt.Run(); err != nil {
				return errors.New("The preflight checks failed")
			}
		}
	} else {
		log.Info("preflight check passed")
	}

	dkim, err := generateDKIM()
	if err != nil {
		return errors.Wrap(err, "could not generate DKIM keys")
	}

	ip, err := GetOutboundIP()
	if err != nil {
		return errors.Wrap(err, "could not get outbound IP")
	}

	base64Dkim := base64.StdEncoding.EncodeToString(dkim)

	if isLocalSetup {
		return setupLocal(ip, base64Dkim)
	} else {
		return setupConnected(ip, base64Dkim)
	}
}
