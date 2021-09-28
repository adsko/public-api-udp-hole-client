package web

import (
	"client/connection"
	"client/hub"
	"client/messages"
	server2 "client/server"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// TODO: Add up to N seconds for message
func Run(server connection.Connection, connectionData *hub.OnConnectionData) error {
	var packetId uint64 = 0
	lock := sync.Mutex{}
	var getAndIncPacketId = func() uint64 {
		lock.Lock()
		defer lock.Unlock()
		packetId++
		return packetId
	}
	binaryClientId := make([]byte, 8)
	binary.LittleEndian.PutUint64(binaryClientId, connectionData.ClientID)
	receivedPing := false

	clientCipher, err := aes.NewCipher(server2.Secret(connectionData.ClientSecret[:]))

	if err != nil {
		return err
	}

	serverCipher, err := aes.NewCipher(server2.Secret(connectionData.ServerSecret[:]))
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
			var message []byte = nil
			if sendOnAddr != nil {
				message = messages.BuildPongMessage(clientCipher, getAndIncPacketId(), addr, sendOnAddr)
			} else {
				message = messages.BuildPongMessage(clientCipher, getAndIncPacketId(), addr, addr)
			}
			err = server.Send(message, addr, binaryClientId)
			if err != nil {
				log.Println(err)
			}
		}
	}

	go sendPong(nil)

	server.OnData(connectionData.ClientID, func(data []byte, port int) {
		if receivedPing {
			log.Println("Skipped PING because received")
			return
		}

		receivedPing = true
		decoded, _, err := messages.Open(serverCipher, data)
		message, err := messages.DecodePingMessage(decoded)
		message.Port = port
		if err != nil {
			log.Println(err)
		} else {
			sendPong(message) // ClientConnection almost done, now need to wait for init message
			go connectionEstablished(server, connectionData, clientCipher, serverCipher, binaryClientId, getAndIncPacketId)
		}
	})

	for {
		time.Sleep(5 * time.Second)
	}
}

func connectionEstablished(server connection.Connection, connectionData *hub.OnConnectionData, clientCipher, serverCipher cipher.Block, binaryClientId []byte, getAndIncPacketId func() uint64) {
	var serverIp *net.UDPAddr = nil
	serverStarted := false
	connections := make(map[uint64]*sender)
	var lastPacketId uint64 = 0
	packetsQueue := make(map[uint64][]byte)
	var handleClientData func(data []byte, port int)
	connectionError := make(chan string)
	ackData := make(map[uint64]uint)
	ackDataLock := new(sync.Mutex)

	handleClientData = func(data []byte, port int) {
		decoded, nonce, err := messages.Open(serverCipher, data)
		if err != nil {
			log.Println("Error when decoding message!")
			return
		}

		if serverIp != nil && messages.GetMessageType(decoded) != messages.ACKType {
			go server.Send(messages.BuildAckMessage(clientCipher, nonce), serverIp, binaryClientId)
		}

		if messages.GetMessageType(decoded) != messages.InitType && messages.GetMessageType(decoded) != messages.ACKType {
			if nonce > lastPacketId+1 {
				packetsQueue[nonce] = data
				return
			}

			if nonce <= lastPacketId {
				return // Duplicate
			}

			lastPacketId = nonce
		}

		switch messages.GetMessageType(decoded) {
		case messages.InitType:
			serverIP, err := messages.DecodeInitMessage(decoded)
			if err != nil {
				log.Panic(err)
			}

			lastPacketId = nonce
			serverIP.Port = port
			serverIp = serverIP
			if serverStarted {
				return
			}

			makeSender := func(connectionNumber uint64) *sender {
				s := &sender{
					s:                 server,
					serverIP:          serverIP,
					serverCipher:      serverCipher,
					clientCipher:      clientCipher,
					clientId:          binaryClientId,
					data:              make(chan []byte),
					closed:            make(chan bool),
					error:             connectionError,
					getAndIncPackedId: getAndIncPacketId,
					ackData:           ackData,
					ackDataLock:       ackDataLock,
				}

				connections[connectionNumber] = s

				return s
			}

			makeSender(0).SendInit()
			go runTCP("9191", &makeSender)
			serverStarted = true
		case messages.PingType:
			// Skip this message

		case messages.EOFType:
			connectionId := messages.DecodeEOFMessage(decoded)
			sender, ok := connections[connectionId]
			if ok {
				sender.closed <- true
			}

		case messages.ErrorType:
			connectionId := messages.DecodeEOFMessage(decoded)
			sender, ok := connections[connectionId]
			if ok {
				sender.closed <- true
			}
		case messages.DataType:
			message, connectionId := messages.DecodeDataMessage(decoded)
			sender, ok := connections[connectionId]
			if ok {
				sender.NewData(message)
			}

		case messages.ACKType:
			messageId := messages.DecodeACKMessage(decoded)
			ackDataLock.Lock()
			delete(ackData, messageId)
			ackDataLock.Unlock()

		default:
			log.Printf("Not defined message type %d!", messages.GetMessageType(decoded))
		}

		nextMessage, hasNextData := packetsQueue[nonce+1]
		if hasNextData {
			delete(packetsQueue, nonce+1)
			handleClientData(nextMessage, port)
		}
	}

	server.OnData(connectionData.ClientID, handleClientData)

	for {
		select {
		case err := <-connectionError:
			log.Panic(err)
		default:
			time.Sleep(time.Second)
		}
	}
}

type sender struct {
	s                 connection.Connection
	serverIP          *net.UDPAddr
	serverCipher      cipher.Block
	clientCipher      cipher.Block
	clientId          []byte
	data              chan []byte
	closed            chan bool
	close             bool
	error             chan string
	getAndIncPackedId func() uint64

	ackData     map[uint64]uint
	ackDataLock *sync.Mutex
}

func (s *sender) Send(data []byte, connectionNumber uint64) {
	messageId := s.getAndIncPackedId()
	message := messages.BuildDataMessage(s.clientCipher, messageId, data, connectionNumber)
	s.InitAck(messageId)
	go s.WaitForAck(message, messageId)
	err := s.s.Send(message, s.serverIP, s.clientId)
	if err != nil {
		log.Println(err)
	}

}

func (s *sender) SendInit() {
	messageId := s.getAndIncPackedId()
	message := messages.BuildInitMessage(s.clientCipher, messageId, s.serverIP)
	s.s.Send(message, s.serverIP, s.clientId)
	s.InitAck(messageId)
	go s.WaitForAck(message, messageId)
}

func (s *sender) InitAck(messageId uint64) {
	s.ackDataLock.Lock()
	defer s.ackDataLock.Unlock()
	s.ackData[messageId] = 0
}

func (s *sender) WaitForAck(message []byte, messageId uint64) {
	time.Sleep(500 * time.Millisecond)

	if s.close {
		return
	}

	s.ackDataLock.Lock()
	currentRetry, ok := s.ackData[messageId]
	if ok {
		s.ackData[messageId] += 1
	}
	s.ackDataLock.Unlock()

	if ok {
		if currentRetry > 5 {
			s.error <- "no connection (ACK)"
			s.close = true
			return
		}

		go s.WaitForAck(message, messageId)
		err := s.s.Send(message, s.serverIP, s.clientId)
		if err != nil {
			log.Println(err)
		}
	}
}

func (s *sender) NewData(data []byte) {
	s.data <- data
}

func runTCP(port string, makeSender *func(uint64) *sender) error {
	p := fmt.Sprintf(":%s", port)
	var connectionNumber uint64 = 0
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

		go handle(acceptedCon, (*makeSender)(connectionNumber), connectionNumber)
		connectionNumber++
	}
}

func handle(conn net.Conn, sender *sender, connectionNumber uint64) {
	defer conn.Close()
	readAll := func() {
		tcpbuffer := make([]byte, 1024*15)

		for {
			size, err := conn.Read(tcpbuffer)
			if err == io.EOF {
				break
			}
			if err == nil {
				sender.Send(tcpbuffer[:size], connectionNumber)
			}
		}
	}

	go readAll()

	for {
		select {
		case data := <-sender.data:
			conn.Write(data)
		case <-sender.closed:
			return
		}
	}
}
