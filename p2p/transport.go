package p2p

// Peer represents as an interface for a remote node
type Peer struct{}

// Transport represents as an interface for a transport layer
type Transport interface {
	listenAndAccept() error
}
