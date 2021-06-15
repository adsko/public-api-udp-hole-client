package stun

import (
	"github.com/pion/stun"
	"net"
	"strconv"
	"sync"
	"time"
)


type udpStunClient struct {
	c *net.UDPConn
}

func (c *udpStunClient) Write(p []byte) (n int, err error) {
	addr, err := net.ResolveUDPAddr("udp","stun.l.google.com:19302")
	if err != nil {
		return
	}

	n, err = c.c.WriteToUDP(p, addr)
	return
}

func (c *udpStunClient) Read(p []byte) (n int, err error) {
	n, err = c.c.Read(p)
	return
}

func (c *udpStunClient) Close() error {
	return nil
}

func GetAllSocketIps() ([]string, error) {
	var s []string
	ifaces, err := net.Interfaces()

	if err != nil {
		return nil, err
	}

	// handle err
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			s = append(s, ip.String())
		}
	}

	return s, nil
}


func GetConnectionIp(conn *net.UDPConn) (string, string, error) {
	u := udpStunClient{
		c: conn,
	}
	c, err := stun.NewClient(&u, stun.WithNoConnClose)

	if err != nil {
		return "", "", err
	}

	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	lock := sync.Mutex{}
	var apiErr error = nil
	ip := ""
	port := ""

	lock.Lock()
	if err := c.Do(message, func(res stun.Event) {
		defer lock.Unlock()
		if res.Error != nil {
			apiErr = res.Error
			return
		}
		// Decoding XOR-MAPPED-ADDRESS attribute from message.
		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(res.Message); err != nil {
			apiErr = err
			return
		}

		ip = xorAddr.IP.String()
		port = strconv.Itoa(xorAddr.Port)
		return
	}); err != nil {
		apiErr = err
		lock.Unlock()
	}

	lock.Lock()
	lock.Unlock()

	closed := make(chan bool)
	go func() {
		c.Close()
		closed <- true
	}()

	time.Sleep(1 * time.Second)
	a := conn.LocalAddr().String()

	addr, err := net.ResolveUDPAddr("udp", a)
	if err != nil {
		return "", "", err
	}

	t, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return "", "", err
	}

	for {
		select {
			case <- closed:
				t.Close()
				return ip, port, apiErr
		default:
			time.Sleep(100 * time.Millisecond)
			_, err = t.Write([]byte("close_stun"))
			if err != nil {
				return "", "", err
			}
		}
	}
}
