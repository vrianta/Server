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
	sMTPConfig  *sMTPConfig
	sender_mail string
}

var Client = &sMTPClient{}

func (s *sMTPClient) InitSMTPClient(host string, port int, username, password string) error {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", address)
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

	auth := smtp.PlainAuth("", username, password, host)
	if err := client.Auth(auth); err != nil {
		fmt.Printf("failed to Authenticate client : %s", err.Error())
		return err
	}

	if err := client.Mail(username); err != nil {
		return err
	}

	s.sMTPConfig = &sMTPConfig{client: client}
	s.sender_mail = username

	return nil
}

func (s *sMTPClient) SendMail(to []string, subject, body string) error {

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

	message := fmt.Sprintf("From: %s\r\nSubject: %s\r\n\r\n%s", s.sender_mail, subject, body)
	_, err = writer.Write([]byte(message))
	return err
}

func (s *sMTPClient) Close() error {
	return s.sMTPConfig.client.Quit()
}