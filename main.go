package main

import (
	"fmt"
	"log"
	"net"

	"github.com/1500-bytes/CAD/p2p"
	"github.com/1500-bytes/CAD/server"
	"github.com/1500-bytes/CAD/store"
)

func initServer(port int, nodes ...int) *server.FileServer {
	transOpts := p2p.TCPTransportOpts{
		ListenAddr: &net.TCPAddr{
			IP:   net.ParseIP("localhost"),
			Port: port,
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
		BootstrapNodes:    nodes,
	}

	s := server.NewFileServer(fileServerOpts)
	return s

}

func main() {

	s1 := initServer(4000, []int{}...)
	s2 := initServer(4001, []int{4000}...)

	go func() {
		if err := s1.Start(); err != nil {
			log.Fatalf("Error starting server: %s", err)
		}
	}()

	s2.Start()
}
