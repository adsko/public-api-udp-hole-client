package server

import (
	"client/connection"
	"client/stun"
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

// TODO: Rename it
// TODO: Move it to new module

type ProxyServer interface {
	Start()
	OnClose() chan bool
	GetAddresses() ([]string, []string, error)
	Connect(secret connection.Secret, ips []connection.IP) (connection.Connection, error)	// TODO: Remove it from proxy
	OnData(clientID uint64, callback func(data []byte))
	Clear(clientID uint64)
	Send(data []byte, ip *net.UDPAddr, clientID []byte) error
}

type proxyServer struct {
	conn *net.UDPConn
	closedChan chan bool
	counter uint64
	dataCallbacks map[uint64]func([]byte)
}

func (p *proxyServer) OnData(clientID uint64, callback func(data []byte)) {
	p.dataCallbacks[clientID] = callback
}

func (p *proxyServer) Clear(clientID uint64) {
	delete(p.dataCallbacks, clientID)
}

func (p *proxyServer) Send(data []byte, ip *net.UDPAddr, clientID []byte) error {
	out := append(clientID, data...)
	_, err :=  p.conn.WriteToUDP(out, ip)

	return err
}

func (p *proxyServer) watchData () {
	defer func() {
		p.closedChan <- true
	}()
	defer p.conn.Close()

	buf := make([]byte, 64 * 1024)

	for {
		n, _, err := p.conn.ReadFromUDP(buf)

		if n > 8 {
			clientIdBytes := buf[:8]
			clientId := binary.LittleEndian.Uint64(clientIdBytes)
			callback, ok := p.dataCallbacks[clientId]
			if !ok {
				log.Printf("Could not find client: %d", clientId)
			} else {
				callback(buf[8:n])
			}
		}
		if err != nil {
			fmt.Println("Error: ", err)
			break
		}

	}
}

func (p *proxyServer) OnClose() chan bool {
	return p.closedChan
}

func (p *proxyServer) Start () {
	p.dataCallbacks = make(map[uint64]func([]byte))
	p.closedChan = make(chan bool)

	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")

	if err != nil {
		log.Panic("closed server", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Running proxy server: %s", conn.LocalAddr().String())

	err = conn.SetReadBuffer(1024 * 1024)
	if err != nil {
		log.Panic(err)
	}
	p.conn = conn
	go p.watchData()
}

func (p *proxyServer) GetAddresses() ([]string, []string, error) {
	addresses := [1]string{""}
	ports := [1]string{""}

	ip, port, err := stun.GetConnectionIp(p.conn)

	if err != nil {
		return nil, nil, err
	}

	addresses[0] = ip
	ports[0] = port

	return addresses[:], ports[:], nil
}

func (p *proxyServer) Connect(secret connection.Secret, ips []connection.IP) (connection.Connection, error) {
	return connection.GetConnection(secret, ips)
}

func (p *proxyServer) send(data [][]byte, addr *net.UDPAddr) {
	for _, d := range data {
		_, err := p.conn.WriteToUDP(d, addr)
		if err != nil {
			log.Println(err)
		}
	}
}