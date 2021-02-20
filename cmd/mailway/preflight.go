package main

import (
	"fmt"
	"net"
	"time"

	"github.com/mailway-app/config"

	"github.com/pkg/errors"
)

func runPreflightChecks() error {
	port := config.CurrConfig.PortFrontlineSMTP
	if ok, _ := testPort(port); !ok {
		return errors.Errorf("port %d appears to be blocked", port)
	}
	return nil
}

func testPort(port int) (bool, error) {
	addr := fmt.Sprintf("portquiz.net:%d", port)
	timeout := 3 * time.Second
	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false, err
	}
	c.Close()
	return true, nil
}
