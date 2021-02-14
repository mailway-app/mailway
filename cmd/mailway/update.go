package main

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
)

func aptInstall(pkg string) error {
	cmd := exec.Command("apt-get", "install", "-y", pkg)
	log.Debugf("running command: %s", cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to update %s", pkg)
	}
	return nil
}

func aptUpdate() error {
	cmd := exec.Command("apt-get", "update")
	log.Debugf("running command: %s", cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to update metadata")
	}
	return nil
}

func update() error {
	if err := aptUpdate(); err != nil {
		log.Error(err)
	}
	if err := aptInstall("mailway"); err != nil {
		log.Error(err)
	}
	for _, service := range SERVICES {
		if service == "mailway-supervisor" {
			continue
		}
		if err := aptInstall(service); err != nil {
			log.Error(err)
		}
	}

	services("restart")
	return nil
}
