package messages

import (
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

func BuildAckMessage(c cipher.Block, messages uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, messages)

	data := []byte{ACKType}
	data = append(data, buf...)

	return SignRandom(c, data)
}

func DecodeACKMessage(decoded []byte) (messages uint64) {
	return binary.LittleEndian.Uint64(decoded[1:])
}

func BuildInitMessage(c cipher.Block, nonce uint64, ip *net.UDPAddr) []byte {
	data := []byte{InitType}
	data = append(data, ip.IP...)

	return Sign(c, nonce, data)
}

func DecodeInitMessage(decoded []byte) (addr *net.UDPAddr, err error) {
	addr = new(net.UDPAddr)

	if len(decoded) != 17 {
		err = errors.New(fmt.Sprintf("ack, expected 17 but got: %d", len(decoded)))
		return
	}

	if decoded[0] != InitType {
		err = errors.New(fmt.Sprintf("not valid message type expected 3 but got %d", decoded[0]))
		return
	}

	addr.IP = decoded[1:17]
	return
}
