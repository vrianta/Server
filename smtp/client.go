package smtp

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
)

type sMTPConfig struct {
	client *smtp.Client
}

type sMTPClient struct {
	sMTPConfig *sMTPConfig
	auth       smtp.Auth
	address    string

	host               string
	port               int
	username, password string
}

var Client = &sMTPClient{}

func (s *sMTPClient) InitSMTPClient(host string, port int, username, password string) error {

	s.address = fmt.Sprintf("%s:%d", host, port)
	s.host = host
	s.port = port
	s.username = username
	s.password = password

	conn, err := net.Dial("tcp", s.address)
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		fmt.Printf("failed to create client : %s", err.Error())
		return err
	}

	// Skip TLS setup to use an unencrypted connection
	if err := client.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		fmt.Printf("failed to start TLS: %s", err.Error())
		return err
	}

	s.auth = smtp.PlainAuth("", username, password, host)
	if err := client.Auth(s.auth); err != nil {
		fmt.Printf("failed to Authenticate client : %s", err.Error())
		return err
	}

	s.sMTPConfig = &sMTPConfig{client: client}

	return nil
}

func (s *sMTPClient) SendMail(to []string, subject, body string) error {

	if s.sMTPConfig.client.Noop() != nil {
		fmt.Printf("failed connection reestablishing")
		if err := s.InitSMTPClient(s.host, s.port, s.username, s.password); err != nil {
			return err
		}
	}
	if err := s.sMTPConfig.client.Mail(s.username); err != nil {
		return err
	}

	for _, recipient := range to {
		if err := s.sMTPConfig.client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := s.sMTPConfig.client.Data()
	if err != nil {
		return err
	}
	defer writer.Close()

	message := fmt.Sprintf("From: %s\r\nSubject: %s\r\n\r\n%s", s.username, subject, body)
	_, err = writer.Write([]byte(message))

	return err

}

func (s *sMTPClient) Close() error {
	return s.sMTPConfig.client.Quit()
}
