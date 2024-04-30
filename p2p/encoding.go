package p2p

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type GOBDecoder struct{}

func (g *GOBDecoder) Decode(r io.Reader, msg *RPC) error {
	fmt.Printf("Decoding: %v\n", msg)
	return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct{}

func (d *DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	peekbuf := make([]byte, 1)
	if _, err := r.Read(peekbuf); err != nil {
		return nil
	}
	stream := peekbuf[0] == IncomingStream
	if stream {
		log.Println("Stream detected")
		msg.Stream = true
		return nil
	}

	buf := make([]byte, 2000)
	n, err := r.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read from connection: %v", err)
	}

	msg.Payload = buf[:n]
	return nil
}

// func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
// 	peekBuf := make([]byte, 1)
// 	if _, err := r.Read(peekBuf); err != nil {
// 		return nil
// 	}

// 	// In case of a stream we are not decoding what is being sent over the network.
// 	// We are just setting Stream true so we can handle that in our logic.
// 	stream := peekBuf[0] == IncomingStream
// 	if stream {
// 		msg.Stream = true
// 		return nil
// 	}

// 	buf := make([]byte, 1028)
// 	n, err := r.Read(buf)
// 	if err != nil {
// 		return err
// 	}

// 	msg.Payload = buf[:n]

// 	return nil
// }
