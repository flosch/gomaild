/*
Gomaild is a tiny mailserver which supports standard unencrypted (no TLS for instance) 
mail transfer and has no support for any mail extensions in existence. It has
no relay capability and handles incoming mails only and forwards them to your
very own mail handler. 
*/
package maild

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/textproto"
	"strings"
)

type Server struct {
	address  string
	hostname string
	handler  func(*Mail)
}

// Creates a new mail server with given address and hostname
func NewMailServer(address string, hostname string) *Server {
	s := &Server{
		address:  address,
		hostname: hostname,
	}
	return s
}

// Listens and receives forever; delivers incoming mails to 
// your handler. ListenAndReceive handles each connection in
// a single Goroutine; therefore it might be a good idea
// to increase GOMAXPROCS for very busy servers.
func (s *Server) ListenAndReceive(handler func(*Mail)) error {
	s.handler = handler
	ln, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error during connection accept:", err)
			continue
		}
		go s.handleConnection(conn)
	}
	panic("unreachable")
}

const (
	awaitingHelo = iota
	awaitingMailFrom
	awaitingRcpt
	awaitingData
)

func respond(conn *textproto.Conn, code int, msg string) error {
	return conn.PrintfLine("%d %s", code, msg)
}

func getParam(line string, cmd string) (string, error) {
	if !strings.HasPrefix(line, cmd) {
		return "", errors.New("Command not found")
	}

	return strings.TrimSpace(line[len(cmd):]), nil
}

func (s *Server) handleConnection(conn net.Conn) {
	state := awaitingHelo
	c := textproto.NewConn(conn)
	mail := NewMail()

	// Say hello
	respond(c, 220, "Hi! You're welcome.")

loop:
	for {
		line, err := c.ReadLine()
		if err != nil {
			conn.Close()
			return
		}
		//log.Printf("line=%s\n", line)

		// Check for quit
		if strings.ToLower(strings.TrimSpace(line)) == "quit" {
			respond(c, 221, "Bye")
			c.Close()
			return
		}

		switch state {
		case awaitingHelo:
			// HELO foobar
			param, err := getParam(line, "HELO ")
			if err != nil {
				respond(c, 502, "HELO awaited.")
				continue loop
			}
			addrs, err := net.LookupHost(param) // FIXME: Look for MX?
			//log.Printf("Looking up %s: %s\n", hostname, addrs)
			if err != nil {
				// Might be a SPAM bot
				respond(c, 451, fmt.Sprintf("Cannot resolve your address (%s)", err))
				c.Close()
				return
			}
			if len(addrs) <= 0 {
				respond(c, 451, "Host found, but no addresses found.")
				c.Close()
				return
			}
			respond(c, 250, s.hostname)
			mail.Hostname = param
			state++
		case awaitingMailFrom:
			param, err := getParam(line, "MAIL FROM:")
			if err != nil {
				respond(c, 502, "MAIL FROM awaited.")
				continue loop
			}
			respond(c, 250, "OK")
			mail.From = param
			state++
		case awaitingRcpt:
			param, err := getParam(line, "RCPT TO:")
			if err != nil {
				respond(c, 502, "RCPT TO awaited.")
				continue loop
			}
			respond(c, 250, "OK")
			mail.Recipients = append(mail.Recipients, param)
			state++
		case awaitingData:
			if strings.TrimSpace(line) != "DATA" {
				respond(c, 502, "DATA awaited")
				continue loop
			}
			respond(c, 354, "End data with <CR><LF>.<CR><LF>")
			lines, err := c.ReadDotLines()
			if err != nil {
				respond(c, 451, "Problem processing your data")
				continue loop
			}
			respond(c, 250, "OK, queued!")
			mail.Data = strings.Join(lines, "\r\n")
			//fmt.Printf("Data\n----------------\n%s\n----------------\n", mail.Data)
			s.handler(mail)
			state = awaitingMailFrom // Go back to MAIL FROM
		default:
			panic("Not implemented")
		}

	}
}
