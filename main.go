package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/1500-bytes/CAD/p2p"
	"github.com/1500-bytes/CAD/server"
	"github.com/1500-bytes/CAD/store"
)

func main() {
	transOpts := p2p.TCPTransportOpts{
		ListenAddr: &net.TCPAddr{
			IP:   net.ParseIP("0.0.0.0"),
			Port: 4000,
		},
		Handshake: p2p.NOPEHandshake,
		Decoder:   &p2p.DefaultDecoder{},
	}
	fmt.Printf("Listening on %s\n", transOpts.ListenAddr.String())

	tcpTransport := p2p.NewTCPTransport(transOpts)

	fileServerOpts := server.FileServerOpts{
		PathTransformFunc: store.CasPathTransformFunc,
		RootStorage:       "4000_network",
		Transport:         tcpTransport,
	}

	s := server.NewFileServer(fileServerOpts)

	// fmt.Printf("FileServer started with root storage at %s\n", s)

	go func() {
		time.Sleep(5 * time.Second)
		s.Stop()
	}()

	log.Println("Starting server")
	if err := s.Start(); err != nil {
		log.Fatalf("Error starting server: %s", err)
	}

}
