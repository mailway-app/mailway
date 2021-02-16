package main

import (
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
)

func runPreflightChecks() error {
	if ok, _ := testPort(25); !ok {
		return errors.New("port 25 appears to be blocked")
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
