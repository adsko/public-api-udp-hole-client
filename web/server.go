package web

import (
	"client/connection"
	"client/hub"
	"client/messages"
	server2 "client/server"
	"crypto/aes"
	"encoding/binary"
	"log"
	"net"
	"time"
)

//type connectBody struct {
//	Server string
//	Addr []string
//	Port string
//}
//
//type connectResponse struct {
//	Addr []string
//	Port string
//}

// TODO: Extract here some kind of abstraction
// TODO: Add up to N seconds for message
func Run(server connection.Connection, connectionData *hub.OnConnectionData) error {
	binaryClientId := make([]byte, 8)
	binary.LittleEndian.PutUint64(binaryClientId, connectionData.ClientID)
	receivedPing := false

	aesBlock, err := aes.NewCipher(server2.Secret(connectionData.Secret)[:])

	if err != nil {
		return err
	}

	sendPong := func(sendOnAddr *net.UDPAddr) {
		for _, ip := range connectionData.IPs {
			addr, err := net.ResolveUDPAddr("udp", string(ip))
			if err != nil {
				continue
			}

			if err != nil {
				log.Panic(err)
			}

			log.Printf("Sending pong to: %s", addr.String())
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
			sendPong(message) // ClientConnection almost done, now need to wait for init message
		}
	})

	mockAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:1")
	sendPong(mockAddr)
	time.Sleep(time.Millisecond * 100)

	for {

	}
}

//func runTCP (port string, handler ClientHandler) error {
//	p := fmt.Sprintf(":%s", port)
//	log.Printf("Running client server on port: %s", p)
//	addr, err := net.ResolveTCPAddr("tcp", p)
//
//	if err != nil {
//		return err
//	}
//
//	conn, err := net.ListenTCP("tcp", addr)
//
//	if err != nil {
//		return err
//	}
//
//	defer conn.Close()
//
//	for {
//		acceptedCon, err := conn.Accept()
//		if err != nil {
//			return err
//		}
//
//		go handle(acceptedCon, handler)
//	}
//}

//func handle (conn net.Conn, handler ClientHandler) {
//	defer conn.Close()
//	// Make a buffer to hold incoming data.
//	netData, err := bufio.NewReader(conn).ReadString('\n')
//
//	if err != nil {
//		log.Println(err)
//		return
//	}
//
//	log.Println(netData)
//}
