package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/1500-bytes/CAD/crypto"
	"github.com/1500-bytes/CAD/p2p"
	"github.com/1500-bytes/CAD/server"
	"github.com/1500-bytes/CAD/store"
)

func initServer(id string, port int, nodes ...int) *server.FileServer {
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
		ID:                id,
		EncKey:            crypto.NewEncryptionKey(),
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
	id := crypto.NewID()

	s1 := initServer(id, 3000, []int{}...)
	s2 := initServer(id, 4000, 3000)
	s3 := initServer(id, 5000, 3000, 4000)

	go func() { log.Fatal(s1.Start()) }()
	time.Sleep(500 * time.Millisecond)
	go func() { log.Fatal(s2.Start()) }()

	time.Sleep(2 * time.Second)

	go s3.Start()
	time.Sleep(2 * time.Second)

	for i := 0; i < 3; i++ {
		log.Printf("Unique ID: %s\n", s3.ID)
		key := fmt.Sprintf("picture_%d.png", i)
		data := bytes.NewReader([]byte("NO BS! Vanilla Golang at its best!"))
		s3.StoreData(key, data)

		if err := s3.Store.Delete(s3.ID, key); err != nil {
			log.Fatal(err)
		}

		r, err := s3.Get(key)
		if err != nil {
			log.Fatal(err)
		}

		b, err := io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(b))
	}
}
