package main

import (
	"fmt"
	"log"

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

	tr := p2p.NewTCPTransport(tcpOptions)

	go func() {
		for {
			msg := <-tr.Consume()
			fmt.Printf("%+v\n", msg)
		}
	}()

	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	select {}
}
