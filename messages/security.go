package messages

import (
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
)

func Sign (aes cipher.Block, nonce uint64, data []byte) []byte {
	buf := make([]byte, 12)
	binary.PutUvarint(buf, nonce)
	aesgcm, _ := cipher.NewGCM(aes)
	var out []byte

	encoded := aesgcm.Seal(out[:0], buf[:], data, nil)

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, nonce)

	return append(b, encoded...)
}

func Open(aes cipher.Block, data []byte) ([]byte, error) {
	if len(data) < 8 {
		return nil, errors.New(fmt.Sprintf("open, not valid message length: %d", len(data)))
	}
	nonce := make([]byte, 12)
	binary.PutUvarint(nonce, binary.LittleEndian.Uint64(data[:8]))
	toDecode := data[8:]
	var out []byte

	aesgcm, _ := cipher.NewGCM(aes)
	out, err := aesgcm.Open(out[:], nonce, toDecode, nil)

	if err != nil {
		return out, err
	}

	return out, nil
}

var counter uint64 = 0
var lock = sync.Mutex{}

func GetAndIncCounter () uint64 {
	lock.Lock()
	defer lock.Unlock()

	counter += 1
	return counter
}