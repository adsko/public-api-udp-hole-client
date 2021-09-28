package messages

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
)

func SignRandom(aes cipher.Block, data []byte) []byte {
	buf := make([]byte, 12)
	var out []byte
	aesgcm, _ := cipher.NewGCM(aes)
	rand.Read(buf)
	encoded := aesgcm.Seal(out[:0], buf[:], data, nil)

	return append(append([]byte{1}, buf...), encoded...)
}

func Sign(aes cipher.Block, nonce uint64, data []byte) []byte {
	buf := make([]byte, 12)
	binary.PutUvarint(buf, nonce)
	aesgcm, _ := cipher.NewGCM(aes)
	var out []byte

	encoded := aesgcm.Seal(out[:0], buf[:], data, nil)

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, nonce)

	return append(append([]byte{0}, b...), encoded...)
}

func Open(aes cipher.Block, data []byte) ([]byte, uint64, error) {
	if len(data) < 8 {
		return nil, 0, errors.New(fmt.Sprintf("open, not valid message length: %d", len(data)))
	}

	messageType := data[0]
	var nonce []byte

	nonce = make([]byte, 12)

	nonceUint64 := binary.LittleEndian.Uint64(data[1:9])
	binary.PutUvarint(nonce, nonceUint64)
	toDecode := data[9:]

	if messageType != 0 {
		nonce = data[1:13]
		toDecode = data[13:]
	}

	var out []byte

	aesgcm, _ := cipher.NewGCM(aes)
	out, err := aesgcm.Open(out[:], nonce, toDecode, nil)

	if err != nil {
		return out, nonceUint64, err
	}

	return out, nonceUint64, nil
}
