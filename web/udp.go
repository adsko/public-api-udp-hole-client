package web

import (
	"client/stun"
	"fmt"
	"log"
	"net"
	"strings"
)

// TODO: Rename it or merge with server

type ClientHandler interface {
	Stop()
}


type clientHandler struct {
	client *net.UDPConn
}

func (u *clientHandler) Stop() {
	u.client.Close()
}

func (u *clientHandler) start (port string) error {
	ip := fmt.Sprintf(":%s", port)
	log.Printf("Staring udp client: %s", ip)
	addr, err := net.ResolveUDPAddr("udp", ip)

	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)

	if err != nil {
		return err
	}

	u.client = conn

	go func () {
		defer conn.Close()
		buf := make([]byte, 1024)
		for {
			n, a, e := conn.ReadFromUDP(buf)
			log.Println("Received ", string(buf[0:n]), " from ", a)

			if e != nil {
				return
			}
		}
	}()

	return nil
}


func (u *clientHandler) getAddr() (string, string, error) {
	addr, port, err := stun.GetConnectionIp(u.client)

	return addr, port, err
}

func IsIPv4(address string) bool {
	return strings.Count(address, ":") < 2
}

func (u *clientHandler) connect(addresses []string, port string) error {
	for _, addr := range addresses {
		if (!IsIPv4(addr)) {
			continue
		}

		a, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", addr, port))

		if err != nil {
			log.Println(err)
			continue
		}

		_, err = u.client.WriteToUDP([]byte("PING"), a)

		if err != nil {
			log.Println(err)
			continue
		}
	}

	return nil
}