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

	s1 := initServer(3000, []int{}...)
	s2 := initServer(4000, 3000)

	go func() {
		log.Fatal(s1.Start())
	}()
	time.Sleep(1 * time.Second)
	go s2.Start()

	time.Sleep(1 * time.Second)
	// for i := 0; i < 1; i++ {
	data := bytes.NewReader([]byte("hello there"))
	key := "myprivatedata"
	err := s2.StoreData(key, data)
	if err != nil {
		log.Print(err)
	}

	err = s2.Store.Delete(key)
	if err != nil {
		log.Print(err)
	}
	time.Sleep(2 * time.Second)

	// // // select {}
	r, err := s2.Get(key)
	if err != nil {
		log.Println(err)
	}

	b, err := io.ReadAll(r)
	// time.Sleep(2 * time.Second)
	if err == nil {
		fmt.Println("Read the following bytes: ", string(b))
	}

}
