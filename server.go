package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/SuperSection/FileStorage/p2p"
)

type FileServerOptions struct {
	ID                string
	EncKey            []byte
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOptions

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	storage *Storage
	quitch  chan struct{}
}

func NewFileServer(opts FileServerOptions) *FileServer {
	storageOpts := StorageOptions{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	if len(opts.ID) == 0 {
		opts.ID = generateID()
	}

	return &FileServer{
		FileServerOptions: opts,
		storage:           NewStorage(storageOpts),
		quitch:            make(chan struct{}),
		peers:             make(map[string]p2p.Peer),
	}
}

func (server *FileServer) broadcast(msg *Message) error {
	/* encode the message */
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	/* send the message to all peers in the network */
	for _, peer := range server.peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	ID   string
	Key  string
	Size int64
}

type MessageGetFile struct {
	ID  string
	Key string
}

func (server *FileServer) Get(key string) (io.Reader, error) {
	if server.storage.Has(server.ID, key) {
		fmt.Printf("[%s] serving file (%s) from local disk\n", server.Transport.ListenAddr(), key)
		_, r, err := server.storage.Read(server.ID, key)
		return r, err
	}

	fmt.Printf("[%s] don't have file (%s) locally, fetching from network...\n", server.Transport.ListenAddr(), key)

	msg := Message{
		Payload: MessageGetFile{
			ID:  server.ID,
			Key: hashKey(key),
		},
	}

	if err := server.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 500)

	for _, peer := range server.peers {
		/* first read the file size so we can limit the amount of bytes
		   that we need from the connection so it will not keep hanging */
		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)

		n, err := server.storage.writeDecrypt(server.EncKey, server.ID, key, io.LimitReader(peer, fileSize))
		if err != nil {
			return nil, err
		}

		fmt.Printf("[%s] received (%d) bytes over the network from (%s): ", server.Transport.ListenAddr(), n, peer.RemoteAddr())

		peer.CloseStream()
	}

	_, r, err := server.storage.Read(server.ID, key)
	return r, err
}

func (server *FileServer) Remove() {}

func (server *FileServer) Store(key string, r io.Reader) error {
	// 1. Store this file to disk
	// 2. Broadcast this file to all known peers in the network

	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	size, err := server.storage.Write(server.ID, key, tee)
	if err != nil {
		return nil
	}

	msg := Message{
		Payload: MessageStoreFile{
			ID:   server.ID,
			Key:  hashKey(key),
			Size: size + 16,
		},
	}

	if err = server.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 5)

	peers := []io.Writer{}
	for _, peer := range server.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2p.IncomingStream})
	n, err := copyEncrypt(server.EncKey, fileBuffer, mw)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] received and written (%d) bytes to disk\n", server.Transport.ListenAddr(), n)

	return nil
}

func (server *FileServer) Stop() {
	close(server.quitch)
}

func (server *FileServer) OnPeer(p p2p.Peer) error {
	server.peerLock.Lock()
	defer server.peerLock.Unlock()

	server.peers[p.RemoteAddr().String()] = p

	log.Printf("connected with remote %s", p.RemoteAddr())
	return nil
}

func (server *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to error user quit action")
		server.Transport.Close()
	}()

	for {
		select {
		case rpc := <-server.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("decoding error: ", err)
			}

			if err := server.handleMessage(rpc.From, &msg); err != nil {
				log.Println("handle message error: ", err)
			}

		case <-server.quitch:
			return
		}
	}
}

func (server *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return server.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return server.handleMessageGetFile(from, v)
	}

	return nil
}

func (server *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !server.storage.Has(msg.ID, msg.Key) {
		return fmt.Errorf("[%s] need to serve file (%s) but it doesn't exist on disk", server.Transport.ListenAddr(), msg.Key)
	}

	fmt.Printf("[%s] serving file (%s) over the network\n", server.Transport.ListenAddr(), msg.Key)

	fileSize, r, err := server.storage.Read(msg.ID, msg.Key)
	if err != nil {
		return err
	}

	if re, ok := r.(io.ReadCloser); ok {
		fmt.Println("closing readCloser")
		defer re.Close()
	}

	peer, ok := server.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not in map", from)
	}

	/* first send "incomingstream" byte to the peer and
	   then we can send the file size as an int64 */
	peer.Send([]byte{p2p.IncomingStream})
	binary.Write(peer, binary.LittleEndian, fileSize)

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written (%d) bytes over the network to %s\n", server.Transport.ListenAddr(), n, from)

	return nil
}

func (server *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := server.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}

	n, err := server.storage.Write(msg.ID, msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes to disk\n", server.Transport.ListenAddr(), n)

	peer.CloseStream()

	return nil
}

func (server *FileServer) bootstrapNetwork() error {
	for _, addr := range server.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			fmt.Printf("[%s] attempting to connect with remote %s\n", server.Transport.ListenAddr(), addr)

			if err := server.Transport.Dial(addr); err != nil {
				log.Println("dial error: ", err)
			}
		}(addr)
	}

	return nil
}

func (server *FileServer) Start() error {
	fmt.Printf("[%s] starting fileserver...\n", server.Transport.ListenAddr())
	if err := server.Transport.ListenAndAccept(); err != nil {
		return err
	}

	server.bootstrapNetwork()

	server.loop()

	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}
