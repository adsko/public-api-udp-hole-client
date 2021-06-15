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
	Addr     []string
	Port     string
	ClientID uint64
	Secret   string
}

type OnConnectionData struct {
	IPs      []utils.IP
	ClientID uint64
	Secret   string
}

type connectResponse struct {
	Addr     []string
	Port     string
	ClientID uint64
	Secret   string
}

type connectBody struct {
	Server string
	Addr   []string
	Port   string
}
