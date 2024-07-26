package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPTransport(t *testing.T) {
	tcpOptions := TCPTransportOptions{
		ListenAddress: ":4001",
		Decoder:       &DefaultDecoder{},
		HandshakeFunc: NOPHandshakeFunc,
	}

	tr := NewTCPTransport(tcpOptions)
	assert.Equal(t, tr.ListenAddress, ":4001")

	// Server
	assert.Nil(t, tr.ListenAndAccept())

	select {}
}