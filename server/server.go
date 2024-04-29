package server

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"log/slog"
	"sync"
	"time"

	"github.com/1500-bytes/CAD/crypto"
	"github.com/1500-bytes/CAD/p2p"
	"github.com/1500-bytes/CAD/store"
)

type FileServerOpts struct {
	EncKey            []byte
	PathTransformFunc store.PathTransformFunc
	RootStorage       string
	Transport         p2p.Transport
	// BootstrapNodes represents the list of nodes to connect to when starting the server
	BootstrapNodes []int
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store    *store.Store
	quitChan chan struct{}
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

type MessageGetFile struct {
	Key string
}

type Message struct {
	Payload any
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := store.StoreOpts{
		Root:              opts.RootStorage,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		quitChan:       make(chan struct{}),
		store:          store.NewStore(storeOpts),
		FileServerOpts: opts,
		peers:          make(map[string]p2p.Peer),
	}
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return fmt.Errorf("err starting server: %s", err)
	}

	if len(s.BootstrapNodes) > 0 {
		s.Bootstrap()
	}
	s.loop()

	return nil
}

func (s *FileServer) Bootstrap() {
	for _, port := range s.BootstrapNodes {
		go func(port int) {
			if err := s.Transport.Dial(port); err != nil {
				log.Printf("Failed to dial to %d: %v", port, err)
			}
		}(port)
	}
}

func (s *FileServer) Stop() {
	close(s.quitChan)
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p

	return nil
}

func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(key) {
		fmt.Println("find the file locally!")
		_, r, err := s.store.Read(key)
		return r, err
	}

	fmt.Println("file not found locally, broadcasting....")

	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	err := s.broadcast(&msg)
	if err != nil {
		return nil, err
	}

	time.Sleep(1 * time.Second)

	for _, peer := range s.peers {
		var fileSize int64
		err := binary.Read(peer, binary.LittleEndian, &fileSize)
		if err != nil {
			log.Printf("err reading file size: %s\n", err)
		}

		n, err := s.store.Write(key, io.LimitReader(peer, fileSize))
		if err != nil {
			log.Printf("err writing to store: %s\n", err)
			return nil, err
		}

		fmt.Printf("[%s] received (%d) bytes over the network from %s", s.Transport.Addr(), n, peer.RemoteAddr())
		peer.CloseStream()
	}

	_, r, err := s.store.Read(key)
	return r, err
}

func (s *FileServer) StoreData(key string, r io.Reader) error {
	var (
		err        error
		fileBuffer = new(bytes.Buffer)
	)

	// tee reader reads from r and writes as copy to fileBuffer and return the reader which holds the original data
	tee := io.TeeReader(r, fileBuffer)
	size, err := s.store.Write(key, tee)
	if err != nil {
		log.Printf("err writing to store: %s\n", err)
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(10 * time.Millisecond)

	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingStream})
		n, err := crypto.CopyEncrypt(s.FileServerOpts.EncKey, fileBuffer, peer)
		if err != nil {
			slog.Error("err in encryption", "err", err)
		}
		slog.Info("Sent the following to peer", "peer", peer.RemoteAddr().String(), "size", n)
	}

	return nil
}

func (s *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return fmt.Errorf("err encoding message: %s", err)
	}

	i := 0
	for _, peer := range s.peers {
		i++
		fmt.Println(i)
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return fmt.Errorf("err sending message: %s", err)
		}
	}
	return nil
}

func (s *FileServer) loop() {
	defer func() {
		log.Println("Shutting down server")
		s.Transport.Close()
	}()

loop:
	for {
		select {
		case rpc := <-s.Transport.Consume():
			var m Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&m); err != nil {
				log.Printf("Context[s.loop()]: Failed to decode payload: %s", err)
				break loop
			}
			if err := s.handleMessage(rpc.From, &m); err != nil {
				log.Println(err)
			}
		case <-s.quitChan:
			break loop
		}
	}
}

func (s *FileServer) handleMessage(from string, m *Message) error {
	switch v := m.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	}
	return nil
}

func (s *FileServer) handleMessageStoreFile(from string, m MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer not found: %s", from)
	}

	if _, err := s.store.Write(m.Key, io.LimitReader(peer, m.Size)); err != nil {
		return fmt.Errorf("err writing to store: %s", err)
	}

	peer.CloseStream()
	return nil
}

// handleMessageGetFile handles the message get file
func (s *FileServer) handleMessageGetFile(from string, m MessageGetFile) error {
	// check if the key exists in the store
	if !s.store.Has(m.Key) {
		return fmt.Errorf("file not found: %s", m.Key)
	}

	fmt.Printf("[%s] file (%s) over the network\n", s.Transport.Addr(), m.Key)
	// read the file from the store
	fileSize, r, err := s.store.Read(m.Key)
	if err != nil {
		return fmt.Errorf("err reading from store: %s", err)
	}

	if rc, ok := r.(io.ReadCloser); ok {
		fmt.Println("closing readCloser")
		defer rc.Close()
	}
	// get the peer from the map
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer not found: %s", from)
	}

	// write the file to the peer
	peer.Send([]byte{p2p.IncomingStream})
	binary.Write(peer, binary.LittleEndian, fileSize)
	n, err := io.Copy(peer, r)
	if err != nil {
		return fmt.Errorf("err writing to peer: %s", err)
	}

	fmt.Printf("[%s] written (%d) bytes over the network from %s", s.Transport.Addr(), n, from)
	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}
