package hub

import (
	"client/utils"
)

type register struct {
	Name     string
	Port     string
	Addr     []string
	Password string
}

type openConnection struct {
	Addr         []string
	Port         string
	ClientID     uint64
	ClientSecret string
	ServerSecret string
}

type OnConnectionData struct {
	IPs          []utils.IP
	ClientID     uint64
	ClientSecret string
	ServerSecret string
}

type connectResponse struct {
	Addr         []string
	Port         string
	ClientID     uint64
	ClientSecret string
	ServerSecret string
}

type connectBody struct {
	Server string
	Addr   []string
	Port   string
}
