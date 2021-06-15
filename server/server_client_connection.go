package server

import (
	"client/messages"
	"client/utils"
	"crypto/aes"
	"crypto/cipher"
	"log"
	"net"
	"time"
)

type ClientConnection interface {
	HandleData(data []byte)
	StartServer(chan string)
	OnSend(func(data []byte, ip *net.UDPAddr) error)
	Stop()
}

type Secret []byte

type clientConnection struct {
	ips      []utils.IP
	secret   cipher.Block
	data     chan []byte
	sendFunc func(data []byte, ip *net.UDPAddr) error
}

func (c *clientConnection) Stop() {
	c.sendFunc = nil
	close(c.data)
}

func (c *clientConnection) start(asServer bool, close chan string) {
	connected := make(chan bool)
	c.connect(connected, asServer, close)

	isConnected := <-connected

	if isConnected {

	}
}

func (c *clientConnection) OnSend(cb func(data []byte, ip *net.UDPAddr) error) {
	c.sendFunc = cb
}

func (c *clientConnection) StartServer(close chan string) {
	c.data = make(chan []byte, 100)
	go c.start(true, close)
}

func (c *clientConnection) HandleData(data []byte) {
	c.data <- data
}

func (c *clientConnection) Send(data []byte, ip *net.UDPAddr) {
	err := c.sendFunc(data, ip)
	if err != nil {
		log.Println(err)
	}
}

func (c *clientConnection) connect(connected chan bool, asServer bool, close chan string) {
	sendCount := 0
	sendData := func() {
		for _, ip := range c.ips {
			addr, err := net.ResolveUDPAddr("udp", string(ip))

			if err != nil {
				log.Println(err)
				continue
			}

			if asServer {
				pingMessage := messages.BuildPingMessage(c.secret, messages.GetAndIncCounter(), addr)
				log.Printf("Sending ping to %s", ip)
				c.Send(pingMessage, addr)
			}

		}
	}

	go sendData()

	for {
		select {
		case receivedData := <-c.data:
			if asServer {
				sendOnIpAddr, receiverIpAddr, err := messages.DecodePongMessage(receivedData, c.secret)
				if err != nil {
					log.Println(err)
				} else {
					emptyIP, _ := net.ResolveUDPAddr("udp", "0.0.0.0:0")

					log.Printf("Received ips: %s, %s", sendOnIpAddr.String(), receiverIpAddr.String())
					if receiverIpAddr.IP.Equal(emptyIP.IP) {
						log.Printf("Ommiting empty IP")
					} else {
						log.Printf("esablished clientConnection: %s | %s", sendOnIpAddr.IP.String(), receiverIpAddr.IP.String())
						goto CONNECTED
					}
				}
				break
			}
		case <-time.After(time.Millisecond * 1000):
			sendCount += 1
			if sendCount >= 5 {
				close <- "Not received pong"
				return
			}
			go sendData()
		}
	}

CONNECTED:
}

func GetClientConnection(secret Secret, ips []utils.IP) (ClientConnection, error) {
	aesBlock, err := aes.NewCipher(secret[:])
	if err != nil {
		return nil, err
	}

	c := clientConnection{
		ips:    ips,
		secret: aesBlock,
	}

	return &c, nil
}
