package messages

import (
	"crypto/cipher"
	"encoding/binary"
)

func BuildDataMessage(c cipher.Block, nonce uint64, dataToSend []byte, connectionNumber uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, connectionNumber)

	data := []byte{DataType}
	data = append(data, b...)
	data = append(data, dataToSend...)

	return Sign(c, nonce, data)
}

func DecodeDataMessage(decoded []byte) ([]byte, uint64) {
	i := binary.LittleEndian.Uint64(decoded[1:9])
	return decoded[9:], i
}

func DecodeEOFMessage(decoded []byte) (connectionNumber uint64) {
	i := binary.LittleEndian.Uint64(decoded[1:9])

	return i
}

func BuildEOFMessage(c cipher.Block, nonce uint64, connectionNumber uint64) []byte {
	data := []byte{EOFType}
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, connectionNumber)
	data = append(data, b...)
	return Sign(c, nonce, data)
}

func BuildErrorMessage(c cipher.Block, nonce uint64, connectionNumber uint64) []byte {
	data := []byte{ErrorType}
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, connectionNumber)
	data = append(data, b...)
	return Sign(c, nonce, data)
}
