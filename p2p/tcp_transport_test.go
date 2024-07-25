package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPTransport(t *testing.T) {
	tcpOptions := TCPTransportOptions{
		ListenAddress: ":5500",
		Decoder:       &GOBDecoder{},
		HandshakeFunc: NOPHandshakeFunc,
	}

	tr := NewTCPTransport(tcpOptions)
	assert.Equal(t, tr.ListenAddress, tr.Decoder, tr.HandshakeFunc)

	// Server
	assert.Nil(t, tr.ListenAndAccept())

	select {}
}