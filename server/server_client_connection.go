package server

import (
	"client/messages"
	"client/utils"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type ClientConnection interface {
	HandleData(data []byte, port int)
	StartServer(string, chan string)
	OnSend(func(data []byte, ip *net.UDPAddr) error)
	Stop()
}

type Secret []byte

type clientConnection struct {
	error        chan string
	closed       bool
	ips          []utils.IP
	clientSecret cipher.Block
	serverSecret cipher.Block
	data         chan struct {
		data []byte
		port int
	}
	sendFunc        func(data []byte, ip *net.UDPAddr) error
	connections     map[uint64]net.Conn
	connectionsLock sync.Mutex

	packetId     uint64
	packetIdLock sync.Mutex

	ackMap     map[uint64]uint
	ackMapLock sync.Mutex
}

func (c *clientConnection) Stop() {
	c.sendFunc = nil
	close(c.data)
}

func (c *clientConnection) start(serverURL string, asServer bool, close chan string) {
	c.connections = make(map[uint64]net.Conn)
	clientIP, _, connected, lastPacketId := c.connect(asServer, close)

	if connected {
		go c.handleConnection(serverURL, close, clientIP, lastPacketId)
	}
}

func (c *clientConnection) sendACK(messageId uint64, clientIp *net.UDPAddr) {
	ackMessage := messages.BuildAckMessage(c.serverSecret, messageId)
	c.Send(ackMessage, clientIp, false)
}

func (c *clientConnection) handleConnection(serverURL string, close chan string, clientIP *net.UDPAddr, connectionPacketId uint64) {
	initialized := false
	var lastPacketId = connectionPacketId
	packetsQueue := make(map[uint64][]byte)

	var handleMessage func(decoded []byte, newPacketId uint64)
	handleMessage = func(decoded []byte, newPacketId uint64) {
		if messages.GetMessageType(decoded) == messages.ACKType {
			c.handleClientData(serverURL, decoded, clientIP)
			return
		}

		if lastPacketId+1 == newPacketId {
			lastPacketId = newPacketId
			c.handleClientData(serverURL, decoded, clientIP)

			queuedPacket, hasNextPacketOnQueue := packetsQueue[newPacketId+1]
			if hasNextPacketOnQueue {
				delete(packetsQueue, newPacketId+1)
				go handleMessage(queuedPacket, newPacketId+1)
			}
		} else {
			if newPacketId <= lastPacketId {
				// Duplicate packet
			} else {
				packetsQueue[newPacketId] = decoded // Put on queue
			}
		}
	}
	for {
		select {
		case data := <-c.data:
			decoded, newPacketId, err := messages.Open(c.clientSecret, data.data)
			if err != nil {
				log.Println(err)
				return
			}

			if messages.GetMessageType(decoded) == messages.InitType {
				if !initialized {
					lastPacketId = newPacketId - 1
					initialized = true
				}
			}

			handleMessage(decoded, newPacketId)

			if messages.GetMessageType(decoded) != messages.ACKType {
				go c.sendACK(newPacketId, clientIP)
			}
		case err := <-c.error:
			close <- err
			return
		default:
			time.Sleep(16 * time.Millisecond)
		}
	}
}

func (c *clientConnection) handleClientData(serverURL string, data []byte, client *net.UDPAddr) {
	switch messages.GetMessageType(data) {
	case messages.DataType:
		message, connectionNumber := messages.DecodeDataMessage(data)

		go c.handleClientDataType(serverURL, message, connectionNumber, client)
	case messages.ACKType:
		c.OnACK(messages.DecodeACKMessage(data))
	default:
		log.Printf("Unknown message type: %d", messages.GetMessageType(data))
	}
}

func (c *clientConnection) handleClientDataType(serverURL string, data []byte, connectionNumber uint64, client *net.UDPAddr) {
	var err error
	c.connectionsLock.Lock()
	conn, ok := c.connections[connectionNumber]
	c.connectionsLock.Unlock()
	if !ok {
		conn, err = net.Dial("tcp", serverURL)

		if err != nil {
			log.Panic(err)
		}

		c.connectionsLock.Lock()
		c.connections[connectionNumber] = conn
		c.connectionsLock.Unlock()

		connectionMessage := func() {
			recvBuf := make([]byte, 1024*10*6)

			for {
				count, err := conn.Read(recvBuf)
				if c.closed {
					conn.Close()
					c.connectionsLock.Lock()
					_, ok := c.connections[connectionNumber]
					if ok {
						delete(c.connections, connectionNumber)
					}
					c.connectionsLock.Unlock()
					return
				}

				if err != nil {
					if err == io.EOF {
						c.Send(messages.BuildEOFMessage(c.serverSecret, c.GetAndIncCounter(), connectionNumber), client, true)
					} else {
						c.Send(messages.BuildErrorMessage(c.serverSecret, c.GetAndIncCounter(), connectionNumber), client, true)
					}
					conn.Close()
					c.connectionsLock.Lock()
					_, ok := c.connections[connectionNumber]
					if ok {
						delete(c.connections, connectionNumber)
					}
					c.connectionsLock.Unlock()
					break
				} else {
					for i := 0; i < count; i += 1024 * 10 {
						c.Send(messages.BuildDataMessage(c.serverSecret, c.GetAndIncCounter(), recvBuf[i:min(i+1024*10, count)], connectionNumber), client, true)
					}
				}
			}
		}

		go connectionMessage()
	}

	_, err = conn.Write(data)

	if err != nil {
		log.Panic(err)
	}
}

func (c *clientConnection) OnSend(cb func(data []byte, ip *net.UDPAddr) error) {
	c.sendFunc = cb
}

func (c *clientConnection) StartServer(serverURL string, close chan string) {
	c.data = make(chan struct {
		data []byte
		port int
	}, 100)
	go c.start(serverURL, true, close)
}

func (c *clientConnection) HandleData(data []byte, port int) {
	c.data <- struct {
		data []byte
		port int
	}{data: data, port: port}
}

func (c *clientConnection) Send(data []byte, ip *net.UDPAddr, handleACK bool) {
	err := c.sendFunc(data, ip)
	if handleACK {
		c.ackMapLock.Lock()
		_, ok := c.ackMap[messages.GetMessageId(data)]
		if !ok {
			c.ackMap[messages.GetMessageId(data)] = 0
		}
		c.ackMapLock.Unlock()
		if messages.GetMessageId(data)%10 == 0 && messages.GetMessageId(data) > 0 {
			c.HandleACK(data, ip)
		} else {
			go c.HandleACK(data, ip)
		}
	}

	if err != nil {
		log.Println(err)
	}
}

func (c *clientConnection) HandleACK(message []byte, ip *net.UDPAddr) {
	if c.closed {
		return
	}

	time.Sleep(time.Millisecond * 100)

	if c.closed {
		return
	}

	c.ackMapLock.Lock()
	if c.closed {
		c.ackMapLock.Unlock()
		return
	}

	ack, ok := c.ackMap[messages.GetMessageId(message)]
	if ok {
		c.ackMap[messages.GetMessageId(message)] += 1
	}
	if ok {
		if ack > 300 {
			c.error <- fmt.Sprintf("no connection %d", messages.GetMessageId(message))
			c.closed = true
			c.ackMapLock.Unlock()
			return
		}
		c.ackMapLock.Unlock()

		if ack%20 == 0 && ack > 0 {
			log.Printf("Resend message (missing ACK) %d (ack retry: %d)", messages.GetMessageId(message), ack)
			c.Send(message, ip, true)
		} else {
			c.HandleACK(message, ip)
		}
		return
	}

	c.ackMapLock.Unlock()
}

func (c *clientConnection) OnACK(messageId uint64) {
	c.ackMapLock.Lock()
	defer c.ackMapLock.Unlock()

	_, ok := c.ackMap[messageId]

	if ok {
		delete(c.ackMap, messageId)
	}
}

func (c *clientConnection) connect(asServer bool, close chan string) (clientIP, serverIP *net.UDPAddr, connected bool, packetId uint64) {
	sendCount := 0
	sendData := func() {
		for _, ip := range c.ips {
			addr, err := net.ResolveUDPAddr("udp", string(ip))

			if err != nil {
				log.Println(err)
				continue
			}

			if asServer {
				pingMessage := messages.BuildPingMessage(c.serverSecret, c.GetAndIncCounter(), addr)
				log.Printf("Sending ping to %s", ip)
				c.Send(pingMessage, addr, false)
			}

		}
	}

	go sendData()

	for {
		select {
		case receivedData := <-c.data:
			if c.closed {
				return
			}

			if asServer {
				decoded, packetId_, err := messages.Open(c.clientSecret, receivedData.data)
				packetId = packetId_
				sendOnIpAddr, receiverFromIpAddr, err := messages.DecodePongMessage(decoded)
				if err != nil {
					log.Println(err)
				} else {
					emptyIP, _ := net.ResolveUDPAddr("udp", "0.0.0.0:0")

					log.Printf("Received ips: %s, %s", sendOnIpAddr.String(), receiverFromIpAddr.String())
					if receiverFromIpAddr.IP.Equal(emptyIP.IP) {
						log.Printf("Ommiting empty IP")
					} else {
						serverIP = sendOnIpAddr
						clientIP = receiverFromIpAddr
						clientIP.Port = receivedData.port
						log.Printf("esablished clientConnection: %s | %s", sendOnIpAddr.IP.String(), receiverFromIpAddr.IP.String())
						goto CONNECTED
					}
				}
				break
			}
		case <-time.After(time.Millisecond * 1000):
			sendCount += 1
			if sendCount >= 5 {
				close <- "Not received pong"
				return nil, nil, false, 0
			}
			go sendData()
		}
	}

CONNECTED:
	selectedClientIp := ""
	for _, ip := range c.ips {
		if strings.Contains(string(ip), clientIP.IP.String()) {
			selectedClientIp = string(ip)
			break
		}
	}

	udpClientIp, err := net.ResolveUDPAddr("udp", selectedClientIp)
	if err != nil {
		log.Println(err)
		return nil, nil, false, packetId
	}
	clientIP = udpClientIp

	time.Sleep(time.Second * 2)
	c.Send(messages.BuildInitMessage(c.serverSecret, c.GetAndIncCounter(), serverIP), udpClientIp, false)
	return clientIP, serverIP, true, packetId
}

func (c *clientConnection) GetAndIncCounter() uint64 {
	c.packetIdLock.Lock()
	defer c.packetIdLock.Unlock()

	c.packetId += 1
	return c.packetId
}

func GetClientConnection(clientSecret, serverSecret Secret, ips []utils.IP) (ClientConnection, error) {
	clientCipher, err := aes.NewCipher(clientSecret[:])
	if err != nil {
		return nil, err
	}

	serverCipher, serverErr := aes.NewCipher(serverSecret[:])
	if serverErr != nil {
		return nil, err
	}

	c := clientConnection{
		ips:             ips,
		clientSecret:    clientCipher,
		serverSecret:    serverCipher,
		connectionsLock: sync.Mutex{},
		packetIdLock:    sync.Mutex{},
		ackMapLock:      sync.Mutex{},
		ackMap:          make(map[uint64]uint),
		error:           make(chan string),
	}

	return &c, nil
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
