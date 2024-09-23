package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"

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

func (server *FileServer) broadcast(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range server.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

type Message struct {
	From    string
	Payload any
}

func (server *FileServer) StoreData(key string, r io.Reader) error {
	// 1. Store this file to disk
	// 2. Broadcast this file to all known peers in the network

	buf := new(bytes.Buffer)
	msg := Message{
		Payload: []byte("storagekey"),
	}
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range server.peers {
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil

	// buf := new(bytes.Buffer)
	// tee := io.TeeReader(r, buf)
	//
	// if err := server.storage.Write(key, tee); err != nil {
	// 	return nil
	// }
	//
	// payload := &MessageData{
	// 	key:  key,
	// 	Data: buf.Bytes(),
	// }
	//
	// return server.broadcast(&Message{
	// 	From:    server.Transport.ListenAddr(),
	// 	Payload: payload,
	// })
}

func (server *FileServer) Stop() {
	close(server.quitch)
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p

	log.Printf("connected with remote %s", p.RemoteAddr())
	return nil
}

func (server *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to user quit action")
		server.Transport.Close()
	}()

	for {
		select {
		case rpc := <-server.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println(err)
			}
			fmt.Printf("receive: %s", string(msg.Payload.([]byte)))
			// if err := server.handleMessage(&m); err != nil {
			// 	log.Println(err)
			// }

		case <-server.quitch:
			return
		}
	}
}

// func (s *FileServer) handleMessage(msg *Message) error {
// 	switch v := msg.Payload.(type) {
// 	case *MessageData:
// 		fmt.Printf("received data %+v\n", v)
// 	}
//
// 	return nil
// }

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

	if len(server.BootstrapNodes) != 0 {

		server.bootstrapNetwork()
	}

	server.loop()

	return nil
}
