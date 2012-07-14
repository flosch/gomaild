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
	address string
	handler func(*Mail)
}

func NewMailServer(address string) *Server {
	s := &Server{
		address: address,
	}
	return s
}

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
	AwaitingHelo = iota
	AwaitingMailFrom
	AwaitingRcpt
	AwaitingData
	AwaitingQuitOrNew
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
	state := AwaitingHelo
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
		case AwaitingHelo:
			// HELO foobar
			if !strings.HasPrefix(line, "HELO") {
				respond(c, 502, "HELO awaited.")
				continue loop
			}
			hostname := strings.TrimSpace(line[5:])
			addrs, err := net.LookupHost(hostname) // FIXME: Look for MX?
			//log.Printf("Looking up %s: %s\n", hostname, addrs)
			if err != nil {
				// Might be a SPAM bot
				respond(c, 451, fmt.Sprintf("Cannot resolve your address (%s)", err))
				c.Close()
				return
			}
			if len(addrs) <= 0 {
				respond(c, 451, "Host found, but not address found.")
				c.Close()
				return
			}
			respond(c, 250, "flosch.dyndns.info")
			mail.Hostname = hostname
			state++
		case AwaitingMailFrom:
			if !strings.HasPrefix(line, "MAIL FROM:") {
				respond(c, 502, "MAIL FROM awaited.")
				continue loop
			}
			from := strings.TrimSpace(line[10:])
			respond(c, 250, "OK")
			mail.From = from
			state++
		case AwaitingRcpt:
			param, err := getParam(line, "RCPT TO:")
			if err != nil {
				respond(c, 502, "RCPT TO awaited.")
				continue loop
			}
			respond(c, 250, "OK")
			mail.Recipients = append(mail.Recipients, param)
			state++
		case AwaitingData:
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
			state = AwaitingMailFrom // Go back to MAIL FROM
		default:
			panic("Not implemented")
		}

	}
}
