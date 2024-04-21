package p2p

import (
	"encoding/gob"
	"fmt"
	"io"
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
	buf := make([]byte, 2000)
	n, err := r.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read from connection: %v", err)
	}

	msg.Payload = buf[:n]
	return nil
}
