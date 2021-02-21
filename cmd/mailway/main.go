package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/mailway-app/config"

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
		"mailway-supervisor",
	}
)

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	if info == nil {
		return false
	}
	return !info.IsDir()
}

func printConfig() {
	s, err := config.PrettyPrint()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", s)
}

// Get preferred outbound ip of this machine
func GetOutboundIP() (*net.IP, error) {
	url := "https://api.ipify.org?format=text"
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call the ip api")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read response body")
	}
	ip := net.ParseIP(string(body[:]))
	return &ip, nil
}

func services(action string) {
	for _, service := range SERVICES {
		cmd := exec.Command("systemctl", action, service)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		log.Debugf("running: %s", cmd)
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
	log.Info("Install certbot")
	cmd := exec.Command("apt-get", "install", "-y", "certbot")
	log.Debug(cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "failed to install certbot")
	}

	log.Info("Run certbot")
	cmd = exec.Command("certbot", "certonly", "--manual",
		"--domain="+config.CurrConfig.InstanceHostname, "--email="+config.CurrConfig.InstanceEmail,
		"--cert-name=smtp-"+config.CurrConfig.InstanceHostname, "--preferred-challenges=dns")
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

func newJWT() error {
	jwt, err := authorize(config.CurrConfig.ServerId)
	if err != nil {
		return errors.Wrap(err, "failed to call authorize")
	}
	token, err := parseJWT(jwt)
	if err != nil {
		return errors.Wrap(err, "failed to parse JWT")
	}
	data, err := getJWTData(token)
	if err != nil {
		return errors.Wrap(err, "failed to get JWT data")
	}
	err = config.WriteInstanceConfig(data.Hostname, data.Email)
	if err != nil {
		return errors.Wrap(err, "failed to write config")
	}
	return nil
}

func main() {
	if err := config.Init(); err != nil {
		log.Fatalf("failed to init config: %s", err)
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("failed to run command: %s", err)
	}
}
