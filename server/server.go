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

// FileServerOpts represents the file server options
type FileServerOpts struct {
	ID                string
	EncKey            []byte
	PathTransformFunc store.PathTransformFunc
	RootStorage       string
	Transport         p2p.Transport
	// BootstrapNodes represents the list of nodes to connect to when starting the server
	BootstrapNodes []int
}

// FileServer represents the file server
type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	Store    *store.Store
	quitChan chan struct{}
}

// MessageStoreFile represents the message to store the file
type MessageStoreFile struct {
	ID   string
	Key  string
	Size int64
}

// MessageGetFile represents the message to get the file
type MessageGetFile struct {
	ID  string
	Key string
}

// MessageDeleteFile represents the message to delete the file
type MessageDeleteFile struct {
	ID  string
	Key string
}

// Message represents the message
type Message struct {
	Payload any
}

// NewFileServer creates a new file server
func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := store.StoreOpts{
		Root:              opts.RootStorage,
		PathTransformFunc: opts.PathTransformFunc,
	}

	if len(opts.ID) == 0 {
		opts.ID = crypto.NewID()
	}

	return &FileServer{
		quitChan:       make(chan struct{}),
		Store:          store.NewStore(storeOpts),
		FileServerOpts: opts,
		peers:          make(map[string]p2p.Peer),
	}
}

// Start starts the server
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

// Bootstrap connects to the bootstrap nodes
func (s *FileServer) Bootstrap() {
	for _, port := range s.BootstrapNodes {
		go func(port int) {
			if err := s.Transport.Dial(port); err != nil {
				log.Printf("Failed to dial to %d: %v", port, err)
			}
		}(port)
	}
}

// Stop stops the server
func (s *FileServer) Stop() {
	close(s.quitChan)
}

// OnPeer adds the peer to the map
func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p

	return nil
}

// Get gets the file from the store
func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.Store.Has(s.ID, key) {
		log.Println("No file found locally")
		_, r, err := s.Store.Read(s.ID, key)
		return r, err
	}

	msg := Message{
		Payload: MessageGetFile{
			ID:  s.ID,
			Key: crypto.HashKey(key),
		},
	}

	slog.Info("No file found locally! broadcasting...", "key", msg.Payload)
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

		n, err := s.Store.WriteDecrypt(s.EncKey, s.ID, key, io.LimitReader(peer, fileSize))
		if err != nil {
			log.Printf("err writing to store: %s\n", err)
			return nil, err
		}

		fmt.Printf("[%s] received (%d) bytes over the network from %s\n", s.Transport.Addr(), n, peer.RemoteAddr())
		peer.CloseStream()
	}

	_, r, err := s.Store.Read(s.ID, key)
	return r, err
}

// StoreData stores the data in the store
func (s *FileServer) StoreData(key string, r io.Reader) error {
	var (
		err        error
		fileBuffer = new(bytes.Buffer)
	)

	// tee reader reads from r and writes as copy to fileBuffer and return the reader which holds the original data
	tee := io.TeeReader(r, fileBuffer)
	size, err := s.Store.Write(s.ID, key, tee)
	if err != nil {
		log.Printf("err writing to store: %s\n", err)
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  crypto.HashKey(key),
			Size: size + 16,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2p.IncomingStream})
	n, err := crypto.CopyEncrypt(s.EncKey, fileBuffer, mw)
	if err != nil {
		slog.Error("err in encryption", "err", err)
	}

	log.Printf("[%s] received and written (%d) bytes to disk\n", s.Transport.Addr(), n)
	return nil
}

// ClearAll clears all the networks
func (s *FileServer) ClearAll(key string) error {
	msg := Message{
		Payload: MessageDeleteFile{
			ID:  s.ID,
			Key: crypto.HashKey(key),
		},
	}

	if s.Store.Has(s.ID, key) {
		if err := s.Store.Delete(s.ID, key); err != nil {
			return fmt.Errorf("err deleting file: %s", err)
		}
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	return nil
}

// broadcast sends the message to all the peers
func (s *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return fmt.Errorf("err encoding message: %s", err)
	}

	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return fmt.Errorf("err sending message: %s", err)
		}
	}
	log.Println("broadcasting done!")
	return nil
}

// loop listens for incoming messages
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

// handleMessage handles the incoming messages
func (s *FileServer) handleMessage(from string, m *Message) error {
	switch v := m.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	case MessageDeleteFile:
		s.handleMessageDeleteFile(from, v)
	}
	return nil
}

// handleMessageStoreFile handles the message store file
func (s *FileServer) handleMessageStoreFile(from string, m MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer not found: %s", from)
	}

	if _, err := s.Store.Write(s.ID, m.Key, io.LimitReader(peer, m.Size)); err != nil {
		return fmt.Errorf("err writing to store: %s", err)
	}

	peer.CloseStream()
	return nil
}

// handleMessageGetFile handles the message get file
func (s *FileServer) handleMessageGetFile(from string, m MessageGetFile) error {
	// check if the key exists in the store
	if !s.Store.Has(s.ID, m.Key) {
		return fmt.Errorf("file not found: %s", m.Key)
	}

	log.Println("Handling GetFile...")

	log.Printf("[%s] file (%s) over the network\n", s.Transport.Addr(), m.Key)
	// read the file from the store
	fileSize, r, err := s.Store.Read(s.ID, m.Key)
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

	log.Printf("[%s] written (%d) bytes over the network from %s\n", s.Transport.Addr(), n, from)
	return nil
}

// handleMessageDeleteFile handles the message delete file
func (s *FileServer) handleMessageDeleteFile(from string, m MessageDeleteFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer not found: %s", from)
	}

	peer.Send([]byte{p2p.IncomingMessage})
	if err := s.Store.Delete(m.ID, m.Key); err != nil {
		return fmt.Errorf("err deleting file: %s", err)
	}

	slog.Info("File deleted", "key", m.Key, "from", from)
	return nil

}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
	gob.Register(MessageDeleteFile{})
}
