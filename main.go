package main

import (
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

const (
	localAddr = "127.0.0.1:53"
	dohURL    = "https:/1.1.1.1/dns-query"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		log.Fatalf("Address couldn't be resolved: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("UDP port cannot be listened (May require root access): %v", err)
	}
	defer conn.Close()

	log.Printf("[*] Go DoH Proxy active: %s -> %s", localAddr, dohURL)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	buf := make([]byte, 512)
	for {
		n, remoteAddr, err := conn.ReadFrom(buf)
		if err != nil {
			continue
		}

		packet := make([]byte, n)
		copy(packet, buf[:n])

		go func(data []byte, addr net.Addr) {
			req, err := http.NewRequest("POST", dohURL, bytes.NewReader(data))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/dns-message")
			req.Header.Set("Accept", "application/dns-message")

			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				respData, err := io.ReadAll(resp.Body)
				if err == nil {
					conn.WriteTo(respData, addr)
				}
			}
		}(packet, remoteAddr)
	}
}
