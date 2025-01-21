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
	s.auth = smtp.PlainAuth("", username, password, host)

	return nil
}

func (s *sMTPClient) SendMail(to []string, subject, body string) error {

	conn, err := net.Dial("tcp", s.address)
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		fmt.Printf("failed to create client : %s", err.Error())
		return err
	}

	// Skip TLS setup to use an unencrypted connection
	if err := client.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		fmt.Printf("failed to start TLS: %s", err.Error())
		return err
	}

	if err := client.Auth(s.auth); err != nil {
		fmt.Printf("failed to Authenticate client : %s", err.Error())
		return err
	}

	if err := client.Mail(s.username); err != nil {
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

	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", s.username, to[0], subject, body)
	_, err = writer.Write([]byte(message))

	return err

}

func (s *sMTPClient) Close() error {
	return s.sMTPConfig.client.Quit()
}
