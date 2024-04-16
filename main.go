package main

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/1500-bytes/CAD/p2p"
)

func main() {

	opts := p2p.TCPTransportOpts{
		ListenAddr: &net.TCPAddr{
			IP:   net.ParseIP("0.0.0.0"),
			Port: 3000,
		},
		Handshake: p2p.NOPEHandshake,
		Decoder:   &p2p.DefaultDecoder{},
		OnPeer: func(p2p.Peer) error {
			fmt.Println("exe")
			return fmt.Errorf("dropping peer")
		},
	}

	t := p2p.NewTCPTransport(opts)

	if err := t.ListenAndAccept(); err != nil {
		slog.Error("Fatal Err: ", err)
	}

	go func(t *p2p.TCPTransport) {
		for {
			msg := <-t.Consume()
			fmt.Printf("Consumed %+v\n", msg)
		}
	}(t)

	select {}
}
