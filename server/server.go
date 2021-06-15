package server

import (
	"client/connection"
	"client/hub"
)

func ConnectToHubAsServer(api, name string, proxy connection.Connection) (hub.ServerHUB, error) {

	hubConnection, err := hub.StartAsServer(proxy, api, name)
	if err != nil {
		return nil, err
	}

	return hubConnection, nil
}

func ConnectToHubAsClient(api, name string, proxy connection.Connection) (*hub.OnConnectionData, error) {
	return hub.StartAsClient(proxy, api, name)
}
