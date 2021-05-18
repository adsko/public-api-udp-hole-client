package main

import (
	"client/connection"
	"client/server"
	"client/web"
	"encoding/binary"
	"log"
	"net"
	"os"
)


/*
Connection:
	TODO: Client has no server address!
	Server sends PING message for all client addresses with address
	Client sends PONG message for all server addresses with 0.0.0.0 address
	Server retries PING messages up to N retires with delay M
	Client receives server PONG message and selects first
	Client sends PONG message with two addresses: selected address and address which used to send
	Server receives messages and selects first

	Connection is established, switch to connection module
 */

func main () {
	switch os.Args[1] {
	case "run-client":
		proxyServer := server.RunProxyServer()
		connectionData, err := server.ConnectToHubAsClient("test", proxyServer)
		if err != nil {
			log.Panic(err)
		}

		web.Run(proxyServer, connectionData)

	case "run-api":
		proxyServer := server.RunProxyServer()

		hub, err := server.ConnectToHubAsServer("test", proxyServer)
		if err != nil {
			log.Panic(err)
		}

		proxyOnClose := proxyServer.OnClose()
		hubOnClose := hub.OnClose()
		for {
			select {
			case <- hubOnClose:
				log.Println("Err: closed hub connection")
				os.Exit(1)
			case <- proxyOnClose:
				log.Println("Err: closed web server")
				os.Exit(1)
			case data := <- hub.OnConnection():
				log.Printf("Client %s is trying to connect, with id %d and secret: %s", data.IPs, data.ClientID, data.Secret)
				handleClient(proxyServer, data)
			}
		}
	}
}

func handleClient (proxyServer server.ProxyServer, data server.OnConnectionData) {
	connection, err := proxyServer.Connect(connection.Secret(data.Secret), data.IPs)
	if err != nil {
		log.Println(err)
	}

	binaryClientId := make([]byte, 8)
	binary.LittleEndian.PutUint64(binaryClientId, data.ClientID)

	proxyServer.OnData(data.ClientID, connection.HandleData)
	connection.OnSend(func(data []byte, ip *net.UDPAddr) error {
		return proxyServer.Send(data, ip, binaryClientId)
	})

	defer func() {
		recoverErr := recover()
		if recoverErr != nil {
			log.Println("Closed connection: ", recoverErr)
			connection.Stop()
			proxyServer.Clear(data.ClientID)
		}
		connection = nil
	}()

	closed := make(chan string)
	connection.StartServer(closed)

	go func () {
		c := <- closed
		log.Printf("Closed connection with: %d, reason: %s", data.ClientID, c)
	}()
}
