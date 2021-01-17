package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"time"

	mconfig "github.com/mailway-app/config"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	API_BASE_URL = "https://apiv1.mailway.app"
)

var (
	SERVICES = []string{
		"mailout",
		"maildb",
		"auth",
		"forwarding",
		"frontline",
	}

	CONFIG *mconfig.Config
)

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func setup() error {
	dkim, err := generateDKIM()
	if err != nil {
		return errors.Wrap(err, "could not generate DKIM keys")
	}

	ip := GetOutboundIP()
	url := fmt.Sprintf(
		"https://dash.mailway.app/helo?server_id=%s&ip=%s&dkim=%s",
		CONFIG.ServerId, ip, url.QueryEscape(base64.StdEncoding.EncodeToString(dkim)))
	fmt.Printf("Open %s\n", url)

	ticker := time.NewTicker(2 * time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			jwt, err := authorize(CONFIG.ServerId)
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
			err = mconfig.WriteInstanceConfig(data.Hostname, data.Email)
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

func printConfig() {
	s, err := mconfig.PrettyPrint()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", s)
}

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "1.1.1.1:53")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

func services(action string) {
	for _, service := range SERVICES {
		log.Debugf("service %s %s", action, service)
		cmd := exec.Command("service", service, action)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			log.Errorf("failed to %s service %s: %s", action, service, err)
		}
	}
}

func logs() {
	args := make([]string, 0)
	args = append(args, "-f")
	for _, service := range SERVICES {
		args = append(args, "-u")
		args = append(args, service)
	}
	cmd := exec.Command("journalctl", args...)
	log.Debugf("running command: %s", cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Errorf("failed to read logs: %s", err)
	}
}

func setupSecureSmtp() error {
	c, err := mconfig.Read()
	if err != nil {
		return errors.Wrap(err, "could not read config")
	}

	log.Info("Install certbot")
	cmd := exec.Command("apt-get", "install", "-y", "certbot")
	log.Debug(cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "failed to install certbot")
	}

	log.Info("Run certbot")
	cmd = exec.Command("certbot", "certonly", "--manual",
		"--domain="+c.InstanceHostname, "--email="+c.InstanceEmail,
		"--cert-name=smtp-"+c.InstanceHostname, "--preferred-challenges=dns")
	log.Debug(cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "failed to install certbot")
	}

	return nil
}

func main() {
	c, err := mconfig.Read()
	if err != nil {
		log.Fatalf("could not read config: %s", err)
	}
	CONFIG = c
	log.SetLevel(c.GetLogLevel())

	switch os.Args[1] {
	case "setup":
		if err := setup(); err != nil {
			log.Fatal(err)
		}
	case "setup-secure-smtp":
		if err := setupSecureSmtp(); err != nil {
			log.Fatal(err)
		}
	case "new-jwt":
		jwt, err := authorize(CONFIG.ServerId)
		if err != nil {
			log.Fatal(err)
		}
		token, err := parseJWT(jwt)
		if err != nil {
			log.Fatal(err)
		}
		data, err := getJWTData(token)
		if err != nil {
			log.Fatal(err)
		}
		err = mconfig.WriteInstanceConfig(data.Hostname, data.Email)
		if err != nil {
			log.Fatal(err)
		}
	case "restart":
		services("restart")
	case "logs":
		logs()
	case "status":
		services("status")
	case "update":
		if err := update(); err != nil {
			log.Fatalf("could not update: %s", err)
		}
	case "config":
		printConfig()
	case "recover":
		if err := recoverEmail(os.Args[2]); err != nil {
			log.Fatalf("could not recover email: %s", err)
		}
	default:
		fmt.Printf("subcommand %s not found\n", os.Args[1])
		os.Exit(1)
	}
}
