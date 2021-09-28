package messages

import (
	"crypto/cipher"
	"errors"
	"fmt"
	"net"
)

func BuildPingMessage(c cipher.Block, nonce uint64, ip *net.UDPAddr) []byte {
	data := []byte{PingType}
	data = append(data, ip.IP...)

	return Sign(c, nonce, data)
}

func BuildPongMessage(c cipher.Block, nonce uint64, sendOnIP, receivedFromIP *net.UDPAddr) []byte {
	data := []byte{PongType}
	data = append(data, sendOnIP.IP...)
	data = append(data, receivedFromIP.IP...)

	return Sign(c, nonce, data)
}

func DecodePingMessage(decoded []byte) (ip *net.UDPAddr, err error) {
	ip = new(net.UDPAddr)

	if len(decoded) != 17 {
		err = fmt.Errorf("ping, expected message length: 17, got: %d", len(decoded))
		return
	}

	if decoded[0] != PingType {
		err = errors.New(fmt.Sprintf("not valid message type expected 1 but got %d", decoded[0]))
		return
	}

	ip.IP = decoded[1:17]
	return
}

func DecodePongMessage(decoded []byte) (sendOnIp, receiverFromIp *net.UDPAddr, err error) {
	sendOnIp = new(net.UDPAddr)
	receiverFromIp = new(net.UDPAddr)

	if len(decoded) != 33 {
		err = fmt.Errorf("pong, expected message length: 33, got: %d", len(decoded))
		return
	}

	if decoded[0] != PongType {
		err = errors.New(fmt.Sprintf("not valid message type expected 2 but got %d", decoded[0]))
		return
	}

	sendOnIp.IP = decoded[1:17]
	receiverFromIp.IP = decoded[17:33]

	return
}
