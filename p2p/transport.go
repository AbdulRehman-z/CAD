package p2p

import "net"

// Peer represents as an interface for a remote node.
type Peer interface{}

// Transport represents as an interface for a transport layer
type Transport interface {
	listenAndAccept() error
}

// Message represents the message that is sent between peers.
type Message struct {
	From net.Addr
	// Payload represents the actual data that is sent between peers.
	Payload []byte
}
