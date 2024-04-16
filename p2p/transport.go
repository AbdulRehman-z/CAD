package p2p

import "net"

// Peer represents as an interface for a remote node.
type Peer interface {
	Close() error
}

// Transport represents as an interface for a transport layer
type Transport interface {
	listenAndAccept() error
	Consume() <-chan RPC
}

// RPC represents the RPC that is sent between peers.
type RPC struct {
	From net.Addr
	// Payload represents the actual data that is sent between peers.
	Payload []byte
}
