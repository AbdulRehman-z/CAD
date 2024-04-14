package main

import (
	"log/slog"
	"net"

	"github.com/1500-bytes/CAD/p2p"
)

func main() {

	t := p2p.NewTCPTransport(&net.TCPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: 3000,
	})

	if err := t.ListenAndAccept(); err != nil {
		slog.Error("Fatal Err: ", err)
	}

	select {}
}
