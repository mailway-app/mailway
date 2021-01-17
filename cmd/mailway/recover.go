package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/mail"
	"net/smtp"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func recoverEmail(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return errors.Wrap(err, "could not read file")
	}

	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		return errors.Wrap(err, "could not read message")
	}

	from := msg.Header.Get("Mw-Int-Mail-From")
	to := msg.Header.Get("Mw-Int-Rcpt-To")

	addr := fmt.Sprintf("127.0.0.1:%d", CONFIG.PortForwarding)
	err = smtp.SendMail(addr, nil, from, []string{to}, data)
	if err != nil {
		return errors.Wrap(err, "could not send email")
	}

	// no failures detected so far means that the message has made it back into
	// the system, we can go ahead and delete the file. If another error occur
	// a new file will be created
	if err := os.Remove(file); err != nil {
		return errors.Wrap(err, "could not delete file")
	}
	log.Info("mail sent")

	return nil
}
