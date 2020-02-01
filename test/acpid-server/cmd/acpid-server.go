package main

import (
	"log"
	"net"
	"os"
)

const socketFile = "/var/run/acpid.socket"

func acpidServer(c net.Conn) {
	log.Printf("Client connected [%s]", c.RemoteAddr().Network())
	c.Write([]byte("power/button PBTN 00000000 00000b31\n"))
	c.Close()
}

func main() {
	log.Println("Starting acpid-server...")
	if err := os.RemoveAll(socketFile); err != nil {
		log.Fatal(err)
	}
	log.Printf("Removed previous socket files at %s...", socketFile)

	l, err := net.Listen("unix", socketFile)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()
	log.Printf("Listening on socket %s", socketFile)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}

		go acpidServer(conn)
	}
}
