package main

import (
	"fmt"
	"log"
	"time"

	"github.com/SuperSection/FileStorage/p2p"
)

func OnPeer(peer p2p.Peer) error {
	// peer.Close()
	fmt.Println("doing some logic with the peer outside of TCP Transport")
	return nil
}

func main() {

	tcpOptions := p2p.TCPTransportOptions{
		ListenAddress: ":3300",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       &p2p.DefaultDecoder{},
		OnPeer:        OnPeer,
	}

	tcpTransport := p2p.NewTCPTransport(tcpOptions)

	fileServerOptions := FileServerOptions{
		StorageRoot:       "3000_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
	}

	fileServer := NewFileServer(fileServerOptions)

	go func() {
		time.Sleep(time.Second * 5)
		fileServer.Stop()
	}()

	if err := fileServer.Start(); err != nil {
		log.Fatal(err)
	}
}
