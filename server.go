package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/SuperSection/FileStorage/p2p"
)

type FileServerOptions struct {
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

	return &FileServer{
		FileServerOptions: opts,
		storage:           NewStorage(storageOpts),
		quitch:            make(chan struct{}),
		peers:             make(map[string]p2p.Peer),
	}
}

func (server *FileServer) stream(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range server.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
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
	Key  string
	Size int64
}

type MessageGetFile struct {
	Key string
}

func (server *FileServer) Get(key string) (io.Reader, error) {
	if server.storage.Has(key) {
		return server.storage.Read(key)
	}

	fmt.Printf("don't have file (%s) locally, fetching from network...\n", key)

	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := server.broadcast(&msg); err != nil {
		return nil, err
	}

	for _, peer := range server.peers {
		fmt.Println("receiving stream from peer: ", peer.RemoteAddr())

		fileBuffer := new(bytes.Buffer)
		n, err := io.Copy(fileBuffer, peer)
		if err != nil {
			return nil, err
		}

		fmt.Println("received bytes over the network: ", n)
		fmt.Println(fileBuffer.String())
	}

	select {}

	return nil, nil
}

func (server *FileServer) Store(key string, r io.Reader) error {
	// 1. Store this file to disk
	// 2. Broadcast this file to all known peers in the network

	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	size, err := server.storage.Write(key, tee)
	if err != nil {
		return nil
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}

	if err = server.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 5)

	// TODO: use a multiwriter here.
	for _, peer := range server.peers {
		peer.Send([]byte{p2p.IncomingStream})
		n, err := io.Copy(peer, fileBuffer)
		if err != nil {
			return err
		}

		fmt.Println("received and written to disk: ", n)
	}

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
	if !server.storage.Has(msg.Key) {
		return fmt.Errorf("need to serve file (%s) but it doesn't exist on disk", msg.Key)
	}

	fmt.Printf("serving file (%s) over the network", msg.Key)

	r, err := server.storage.Read(msg.Key)
	if err != nil {
		return err
	}

	peer, ok := server.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not in map", from)
	}

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("written %d bytes over the network to %s\n", n, from)

	return nil
}

func (server *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := server.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}

	n, err := server.storage.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes to disk\n", server.Transport.ListenAddr(), n)

	// peer.(*p2p.TCPPeer).Wg.Done()
	peer.CloseStream()

	return nil
}

func (server *FileServer) bootstrapNetwork() error {
	for _, addr := range server.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			fmt.Println("attempting to connect with remote: ", addr)

			if err := server.Transport.Dial(addr); err != nil {
				log.Println("dial error: ", err)
			}
		}(addr)
	}

	return nil
}

func (server *FileServer) Start() error {
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
