package web

import (
	"bufio"
	"client/connection"
	"client/messages"
	"client/server"
	"crypto/aes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

type connectBody struct {
	Server string
	Addr []string
	Port string
}

type connectResponse struct {
	Addr []string
	Port string
}


// TODO: Extract here some kind of abstraction
// TODO: Add up to N seconds for message
func Run (server server.ProxyServer, connectionData *server.OnConnectionData) error {
	binaryClientId := make([]byte, 8)
	binary.LittleEndian.PutUint64(binaryClientId, connectionData.ClientID)
	receivedPing := false

	aesBlock, err := aes.NewCipher(connection.Secret(connectionData.Secret)[:])

	if err != nil {
		return err
	}

	sendPong := func (sendOnAddr *net.UDPAddr) {
		for _, ip := range connectionData.IPs {
			addr, err := net.ResolveUDPAddr("udp", string(ip))
			if err != nil {
				continue
			}

			if err != nil {
				log.Panic(err)
			}

			message := messages.BuildPongMessage(aesBlock, messages.GetAndIncCounter(), addr, sendOnAddr)
			err = server.Send(message, addr, binaryClientId)
			if err != nil {
				log.Println(err)
			}
		}
	}

	server.OnData(connectionData.ClientID, func(data []byte) {
		if receivedPing {
			log.Println("Skipped PING because received")
			return
		}

		receivedPing = true
		message, err := messages.DecodePingMessage(data, aesBlock)
		if err != nil {
			log.Println(err)
		} else {
			sendPong(message) 	// Connection almost done, now need to wait for init message
		}
	})

	mockAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:1")
	sendPong(mockAddr)

	for {

	}
}

func runTCP (port string, handler ClientHandler) error {
	p := fmt.Sprintf(":%s", port)
	log.Printf("Running client server on port: %s", p)
	addr, err := net.ResolveTCPAddr("tcp", p)

	if err != nil {
		return err
	}

	conn, err := net.ListenTCP("tcp", addr)

	if err != nil {
		return err
	}

	defer conn.Close()

	for {
		acceptedCon, err := conn.Accept()
		if err != nil {
			return err
		}

		go handle(acceptedCon, handler)
	}
}

func handle (conn net.Conn, handler ClientHandler) {
	defer conn.Close()
	// Make a buffer to hold incoming data.
	netData, err := bufio.NewReader(conn).ReadString('\n')

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(netData)
}