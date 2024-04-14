package p2p

// HandshakeFunc is a function that performs a handshake with a remote peer.
type HandshakeFunc func(Peer) error

// NOPEHandshake is a no-op(means no operation) handshake function that does nothing.
func NOPEHandshake(Peer) error { return nil }
