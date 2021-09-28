package messages

import "encoding/binary"

func GetMessageType(decoded []byte) int {
	return int(decoded[0])
}

func GetMessageId(data []byte) uint64 {
	if data[0] != 0 {
		panic("cant read message id")
	}

	return binary.LittleEndian.Uint64(data[1:9])
}
