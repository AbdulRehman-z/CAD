package p2p

import "net"

// Peer represents as an interface for a remote node.
type Peer interface {
	net.Conn
	Send(b []byte) error
	CloseStream()
}

// Transport represents as an interface for a transport layer
type Transport interface {
	Addr() string
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
	Dial(int) error
}
