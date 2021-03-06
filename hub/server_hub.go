package hub

import (
	proxy2 "client/connection"
	"client/stun"
	"client/utils"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"time"
)

type ServerHUB interface {
	OnClose() chan bool
	OnConnection() chan OnConnectionData
}

type serverHUB struct {
	conn       *websocket.Conn
	closedConn chan bool
	connection chan OnConnectionData
	proxy      proxy2.Connection
}

func (h *serverHUB) handleHubMessages() {
	defer h.conn.Close()
	for {
		_, message, err := h.conn.ReadMessage()
		if err != nil {
			h.closedConn <- true
			return
		}

		c := openConnection{}
		err = json.Unmarshal(message, &c)
		if err != nil {
			h.closedConn <- true
			log.Panic(err)
		}

		var data []utils.IP
		for _, addr := range c.Addr {
			data = append(data, utils.IP(fmt.Sprintf("%s:%s", addr, c.Port)))
		}

		h.connection <- OnConnectionData{
			IPs:          data,
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			ServerSecret: c.ServerSecret,
		}
	}
}

func (h *serverHUB) startAsServer(rendezvousAPI, name string, secure bool) error {
	h.closedConn = make(chan bool)
	h.connection = make(chan OnConnectionData)

	scheme := "ws"
	if secure {
		scheme = "wss"
	}

	u := url.URL{Scheme: scheme, Host: rendezvousAPI, Path: "/register"}

	log.Printf("Connecting to rendezvousAPI: %s", rendezvousAPI)

	connection, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	h.conn = connection

	go h.handleHubMessages()

	ips, ports, err := h.proxy.GetAddresses()

	if err != nil {
		return err
	}

	networksIps, err := stun.GetAllSocketIps()
	if err != nil {
		return err
	}

	networksIps = append(networksIps, ips[0])

	registerData := &register{
		Name: name,
		Port: ports[0],
		Addr: networksIps,
	}

	err = connection.WriteJSON(registerData)

	sendPing := func() {
		for {
			time.Sleep(time.Second * 5)
			h.conn.WriteJSON(registerData)
		}
	}

	if err == nil {
		go sendPing()
	}

	return err
}

func (h *serverHUB) OnConnection() chan OnConnectionData {
	return h.connection
}

func (h *serverHUB) OnClose() chan bool {
	return h.closedConn
}

func StartAsServer(proxy proxy2.Connection, rendezvousAPI, name string, secure bool) (ServerHUB, error) {
	hub := serverHUB{
		proxy: proxy,
	}

	err := hub.startAsServer(rendezvousAPI, name, secure)

	if err != nil {
		return nil, err
	}

	return &hub, nil
}
