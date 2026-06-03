package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type State int

const (
	Greet State = iota
	Helo
	Mail
	Rcpt
	Data
)

func getCommand(line string) string {
	switch {
	case strings.HasPrefix(line, "HELO"):
		return "HELO"

	case strings.HasPrefix(line, "EHLO"):
		return "EHLO"

	case strings.HasPrefix(line, "MAIL FROM"):
		return "MAIL FROM"

	case strings.HasPrefix(line, "RCPT TO"):
		return "RCPT TO"

	case strings.HasPrefix(line, "DATA"):
		return "DATA"

	case strings.HasPrefix(line, "QUIT"):
		return "QUIT"

	case strings.HasPrefix(line, "RSET"):
		return "RSET"
	}

	return ""
}

func handleConnection(c net.Conn) {
	defer c.Close()

	var state State = Greet
	c.Write([]byte("220 wnadzahari.online ESMTP read\r\n"))
	scanner := bufio.NewScanner(c)

	var from string
	var to strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		cmd := getCommand(line)
		switch cmd {
		case "HELO":
			if state != Greet {
				c.Write([]byte("503 Bad sequence\r\n"))
				return
			}

			c.Write([]byte("250 wnadzahari.online\r\n"))
			state = Helo

		case "EHLO":
			if state != Greet {
				c.Write([]byte("503 Bad sequence\r\n"))
				return
			}

			c.Write([]byte("250-wnadzahari.online\r\n"))
			c.Write([]byte("250 HELP\r\n"))
			state = Helo

		case "MAIL FROM":
			if state != Helo {
				c.Write([]byte("503 Bad sequence\r\n"))
				return
			}

			c.Write([]byte("250 Getting to know you\r\n"))
			from = strings.TrimPrefix(line, "MAIL FROM:")
			state = Mail
		case "RCPT TO":
			if state != Mail && state != Rcpt {
				c.Write([]byte("503 Bad sequence\r\n"))
				return
			}

			to.WriteString(strings.TrimPrefix(line, "RCPT TO:"))
			to.WriteString(" ")
			c.Write([]byte("250 Who are you trying to connect to?\r\n"))
			state = Rcpt

		case "QUIT":
			c.Write([]byte("221 Bye\r\n"))
			return

		case "RSET":
			state = Greet
			from = ""
			to.Reset()
			c.Write([]byte("250 OK\r\n"))

		case "DATA":
			if state != Rcpt {
				c.Write([]byte("503 Bad sequence\r\n"))
				return
			}

			c.Write([]byte("354 Start mail input\r\n"))
			state = Data

			var body strings.Builder
			for scanner.Scan() {
				line := scanner.Text()
				if line == "." {
					break
				}
				if strings.HasPrefix(line, "..") {
					line = line[1:]
				}
				body.WriteString(line)
				body.WriteString("\r\n")
			}

			fileCount := 0
			folderPath := filepath.Join("mails")
			entries, err := os.ReadDir(folderPath)
			if err == nil {
				for range entries {
					fileCount++
				}
			}

			fileName := strconv.Itoa(fileCount) + " - " + time.Now().UTC().Format(time.RFC3339)
			fullPath := filepath.Join(folderPath, fileName)

			err = os.MkdirAll(folderPath, 0755)
			if err != nil {
				log.Print("Fail to create folder ", err)
				return
			}

			file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				log.Print("Fail to create file: ", err)
				return
			}
			defer file.Close()

			file.WriteString("=====FROM======\n")
			file.WriteString("From: " + from + "\n")
			file.WriteString("=====TO======\n")
			file.WriteString("To: " + to.String() + "\n")
			file.WriteString("=====BODY======\n")
			file.WriteString("\n" + body.String())
			file.WriteString("=====END======\n")

			c.Write([]byte("250 OK\r\n"))

			from = ""
			to.Reset()
			state = Helo

			log.Println("Done.")

		default:
			c.Write([]byte("500 Unrecognized command\r\n"))
		}

	}
	if err := scanner.Err(); err != nil {
		log.Print("An error occured while scanning the buffer.", err)
	}
}

func main() {
	listener, err := net.Listen("tcp", ":25")
	if err != nil {
		log.Fatal("Error creating listener.", err)
	}
	log.Println("TCP established. Listening on port 25...")
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print("Accept error: ", err)
			continue
		}

		log.Println("Accepting connections...")
		go handleConnection(conn)
	}
}

// sudo setcap 'cap_net_bind_service=+ep' ./smtp

// swaks --to ari@localhost --from swak@localhost --port 25
