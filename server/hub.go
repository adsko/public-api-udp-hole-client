package server

import (
	"bytes"
	"client/connection"
	"client/stun"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"net/url"
)

// TODO: Move it to new module: HUB
// TODO: Separate hub on: ClientHUB and ServerHUB

type OnConnectionData struct {
	IPs []connection.IP
	ClientID uint64
	Secret string
}

type Hub interface {
	StartAsServer(host, name string) error
	StartAsClient(host, name string) (*OnConnectionData, error)
	OnClose() chan bool
	OnConnection() chan OnConnectionData
}

type hub struct {
	conn *websocket.Conn
	closedConn chan bool
	connection chan OnConnectionData
	proxy ProxyServer
}

type connectResponse struct {
	Addr []string
	Port string
	ClientID uint64
	Secret string
}

type connectBody struct {
	Server string
	Addr []string
	Port string
}

func (h *hub) handleHubMessages() {
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

		var data []connection.IP
		for _, addr := range c.Addr {
			data = append(data, connection.IP(fmt.Sprintf("%s:%s", addr, c.Port)))
		}

		h.connection <- OnConnectionData{
			IPs: data,
			ClientID: c.ClientID,
			Secret: c.Secret,
		}
	}
}

func (h *hub) StartAsClient(host, name string) (*OnConnectionData, error) {
	ips, ports, err := h.proxy.GetAddresses()

	if err != nil {
		return nil, err
	}

	networksIps, err := stun.GetAllSocketIps()
	if err != nil {
		return nil, err
	}

	networksIps = append(networksIps, ips[0])

	body := connectBody{
		Server: name,
		Addr: networksIps,
		Port: ports[0],
	}

	data, err := json.Marshal(body)

	if err != nil {
		return nil, err
	}

	u := url.URL{Scheme: "http", Host: host, Path: "/connect"}
	r, err := http.Post(u.String(), "application/json", bytes.NewBuffer(data))

	if err != nil {
		return nil, err
	}

	connectBody, err := io.ReadAll(r.Body)

	if err != nil {
		return nil, err
	}

	serverData := &connectResponse{}
	err = json.Unmarshal(connectBody, serverData)

	if err != nil {
		return nil, err
	}

	var mappedIPs []connection.IP
	for _, addr := range serverData.Addr {
		mappedIPs = append(mappedIPs, connection.IP(fmt.Sprintf("%s:%s", addr, serverData.Port)))
	}

	return &OnConnectionData{
		IPs: mappedIPs,
		ClientID: serverData.ClientID,
		Secret: serverData.Secret,
	}, nil
}

func (h *hub) StartAsServer(host, name string) error {
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

func (h *hub) OnConnection() chan OnConnectionData {
	return h.connection
}

func (h *hub) OnClose() chan bool {
	return h.closedConn
}