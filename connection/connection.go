package connection

import (
	"client/stun"
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

type Connection interface {
	Start()
	OnClose() chan bool
	GetAddresses() ([]string, []string, error)
	OnData(clientID uint64, callback func(data []byte, port int))
	Clear(clientID uint64)
	Send(data []byte, ip *net.UDPAddr, clientID []byte) error
}

type connection struct {
	conn          *net.UDPConn
	closedChan    chan bool
	counter       uint64
	dataCallbacks map[uint64]func([]byte, int)
}

func (p *connection) OnData(clientID uint64, callback func(data []byte, port int)) {
	p.dataCallbacks[clientID] = callback
}

func (p *connection) Clear(clientID uint64) {
	delete(p.dataCallbacks, clientID)
}

func (p *connection) Send(data []byte, ip *net.UDPAddr, clientID []byte) error {
	out := append(clientID, data...)
	_, err := p.conn.WriteToUDP(out, ip)

	return err
}

func (p *connection) watchData() {
	defer func() {
		p.closedChan <- true
	}()
	defer p.conn.Close()

	for {
		// TODO: This buff should be handled better
		buf := make([]byte, 64*1024)

		n, addr, err := p.conn.ReadFromUDP(buf)

		if n > 8 {
			clientIdBytes := buf[:8]
			clientId := binary.LittleEndian.Uint64(clientIdBytes)
			callback, ok := p.dataCallbacks[clientId]
			if !ok {
			} else {
				callback(buf[8:n], addr.Port)
			}
		}
		if err != nil {
			fmt.Println("Error: ", err)
			break
		}

	}
}

func (p *connection) OnClose() chan bool {
	return p.closedChan
}

func (p *connection) Start() {
	p.dataCallbacks = make(map[uint64]func([]byte, int))
	p.closedChan = make(chan bool)

	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")

	if err != nil {
		log.Panic("closed server", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Running connection server: %s", conn.LocalAddr().String())

	err = conn.SetReadBuffer(1024 * 1024)
	if err != nil {
		log.Panic(err)
	}
	p.conn = conn
	go p.watchData()
}

func (p *connection) GetAddresses() ([]string, []string, error) {
	addresses := [1]string{""}
	ports := [1]string{""}

	ip, port, err := stun.GetConnectionIp(p.conn, "stun.l.google.com:19302")

	if err != nil {
		return nil, nil, err
	}

	_, port2, err2 := stun.GetConnectionIp(p.conn, "stun3.l.google.com:19302")

	if err2 != nil {
		return nil, nil, err2
	}

	if port2 != port {
		log.Println("NAT random port definition!")
	}

	addresses[0] = ip
	ports[0] = port

	log.Printf("Output port: %s", port)

	return addresses[:], ports[:], nil
}

func (p *connection) send(data [][]byte, addr *net.UDPAddr) {
	for _, d := range data {
		_, err := p.conn.WriteToUDP(d, addr)
		if err != nil {
			log.Println(err)
		}
	}
}

func RunProxyServer() Connection {
	proxy := connection{}
	proxy.Start()

	return &proxy
}
