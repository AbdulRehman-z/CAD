package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

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
		RootStorage:       strconv.Itoa(port) + "_network",
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	s := server.NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = s.OnPeer
	return s
}

func main() {

	s1 := initServer(3000, []int{}...)
	s2 := initServer(4000, 3000)

	go func() {
		log.Fatal(s1.Start())
	}()
	time.Sleep(1 * time.Second)
	go s2.Start()

	time.Sleep(1 * time.Second)

	bytes := bytes.NewReader([]byte("hlo"))

	err := s2.StoreData("myprivatedata", bytes)

	if err != nil {
		log.Fatalf("Failed to store data: %v", err)
	}

	select {}
}
