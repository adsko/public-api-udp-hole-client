package hub

import (
	"bytes"
	proxy2 "client/connection"
	"client/stun"
	"client/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type ClientHub interface {
}

type clientHUB struct {
	proxy proxy2.Connection
}

func (h *clientHUB) StartAsClient(host, name string, secure bool) (*OnConnectionData, error) {
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
		Addr:   networksIps,
		Port:   ports[0],
	}

	data, err := json.Marshal(body)

	if err != nil {
		return nil, err
	}

	scheme := "http"
	if secure {
		scheme = "https"
	}

	u := url.URL{Scheme: scheme, Host: host, Path: "/connect"}
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

	var mappedIPs []utils.IP
	for _, addr := range serverData.Addr {
		mappedIPs = append(mappedIPs, utils.IP(fmt.Sprintf("%s:%s", addr, serverData.Port)))
	}

	return &OnConnectionData{
		IPs:          mappedIPs,
		ClientID:     serverData.ClientID,
		ClientSecret: serverData.ClientSecret,
		ServerSecret: serverData.ServerSecret,
	}, nil
}

func StartAsClient(proxy proxy2.Connection, host, name string, secure bool) (*OnConnectionData, error) {
	h := clientHUB{
		proxy: proxy,
	}

	return h.StartAsClient(host, name, secure)
}
