package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func init() {
	log.SetFlags(log.Llongfile)
}

func expiryHandler(keyring agent.Agent) {
	for {
		keys, _ := keyring.List()
		for _, key := range keys {
			// determine if the key is a certificate
			if !strings.Contains(key.Format, "cert") {
				continue
			}

			certInt, _, _, _, parseCertError := ssh.ParseAuthorizedKey([]byte(key.String()))
			if parseCertError != nil {
				log.Printf("Failed to parse certificate key: %s", parseCertError)
				continue
			}
			cert, ok := certInt.(*ssh.Certificate)
			if !ok || cert == nil {
				log.Printf("Failed to parse certificate: %#v", certInt)
				continue
			}

			// convert the validBefore and validAfter to time.Time
			validBefore := time.Unix(int64(cert.ValidBefore), 0)
			validAfter := time.Unix(int64(cert.ValidAfter), 0)

			// TODO: make this more fancy
			if validBefore.Year() == 1970 || validAfter.Year() == 1970 {
				continue
			}

			if time.Now().After(validBefore) || time.Now().Before(validAfter) {
				log.Println("Removing expired certificate", key.Comment)
				keyring.Remove(key)
			}
		}
		<-time.After(1 * time.Minute)
	}
}

func main() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	keyring := agent.NewKeyring()

	addr, ok := os.LookupEnv("SSH_AUTH_SOCK")
	if !ok {
		addr = fmt.Sprintf("%s/%d-ssh-agent.socket", os.TempDir(), os.Getuid())
	}

	listener, err := net.Listen("unix", addr)
	if err != nil {
		log.Fatalf("Failed to listen on socket: %s", err)
	}
	defer listener.Close()

	log.Println("SSH Agent started on", listener.Addr().String())

	go expiryHandler(keyring)

	// Serve the SSH agent
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatalf("Failed to accept connection: %s", err)
			}
			serveError := agent.ServeAgent(keyring, conn)
			if errors.Is(serveError, io.EOF) {
				continue
			}
			if serveError != nil {
				log.Printf("Failed to serve agent: %#v", serveError)
			}
		}
	}()

	<-sig
	log.Println("SSH Agent stopped")
}
