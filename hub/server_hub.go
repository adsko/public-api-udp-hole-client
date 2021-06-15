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
			IPs:      data,
			ClientID: c.ClientID,
			Secret:   c.Secret,
		}
	}
}

func (h *serverHUB) startAsServer(host, name string) error {
	h.closedConn = make(chan bool)
	h.connection = make(chan OnConnectionData)
	u := url.URL{Scheme: "ws", Host: host, Path: "/register"}

	log.Printf("Connecting to HUB: %s", host)

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

	return err
}

func (h *serverHUB) OnConnection() chan OnConnectionData {
	return h.connection
}

func (h *serverHUB) OnClose() chan bool {
	return h.closedConn
}

func StartAsServer(proxy proxy2.Connection, host, name string) (ServerHUB, error) {
	hub := serverHUB{
		proxy: proxy,
	}

	err := hub.startAsServer(host, name)

	if err != nil {
		return nil, err
	}

	return &hub, nil
}
