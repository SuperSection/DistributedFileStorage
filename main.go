package main

import (
	"log"

	"github.com/SuperSection/FileStorage/p2p"
)

func main() {
	tcpOptions := p2p.TCPTransportOptions{
		ListenAddress: ":3300",
		Decoder:       &p2p.DefaultDecoder{},
		HandshakeFunc: p2p.NOPHandshakeFunc,
	}

	tr := p2p.NewTCPTransport(tcpOptions)

	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	select {}
}
