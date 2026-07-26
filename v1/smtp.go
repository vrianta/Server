package agai

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	log "github.com/vrianta/gulog"
)

type (
	sMTPConfig struct {
		client *smtp.Client
	}
	SMTP struct {
		// For local use
		sMTPConfig  *sMTPConfig
		auth        smtp.Auth
		address     string
		initialised bool

		// for initialise public use
		Host               string
		Port               int
		Username, Password string
		UseTLS             bool // true for certification varification

	}
)

func (s *SMTP) init() error {

	if s.Host == "" || s.Username == "" || s.Password == "" {
		log.Debug("SMTP config is not settedup so we are moving forward without it")
		return fmt.Errorf("UserName, Password, or Host is missing")
	}

	s.address = fmt.Sprintf("%s:%d", s.Host, s.Port)
	s.auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	s.initialised = true

	return nil
}

func (s *SMTP) SendMail(to []string, subject, body string) error {

	if !s.initialised {
		if err := s.init(); err != nil {
			return err
		}
	}

	conn, err := net.Dial("tcp", s.address)
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		fmt.Printf("failed to create client : %s", err.Error())
		return err
	}

	// Skip TLS setup to use an unencrypted connection
	if err := client.StartTLS(&tls.Config{InsecureSkipVerify: !s.UseTLS}); err != nil {
		log.Error("failed to start TLS: %s", err.Error())
		return err
	}

	if err := client.Auth(s.auth); err != nil {
		log.Error("failed to Authenticate client : %s", err.Error())
		return err
	}

	if err := client.Mail(s.Username); err != nil {
		return err
	}

	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	defer writer.Close()

	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", s.Username, strings.Join(to, ","), subject, body)
	_, err = writer.Write([]byte(message))

	return err

}

func (s *SMTP) Close() error {
	if s.sMTPConfig == nil {
		return fmt.Errorf("smtp client not initialised")
	}
	return s.sMTPConfig.client.Quit()
}
