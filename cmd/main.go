package main

import (
	"client/connection"
	"client/hub"
	"client/server"
	"client/web"
	"encoding/binary"
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
	"net"
	"os"
)

/*
ClientConnection:
	TODO: Client has no server address!
	Server sends PING message for all client addresses with address
	Client sends PONG message for all server addresses with 0.0.0.0 address
	Server retries PING messages up to N retires with delay M
	Client receives server PONG message and selects first
	Client sends PONG message with two addresses: selected address and address which used to send
	Server receives messages and selects first

	ClientConnection is established, switch to connection module
*/

var (
	app = kingpin.New("hole-app", "")
	api = app.Flag("api", "Address of the API server").Required().String()

	runClient = app.Command("run-client", "Run client")
	runServer = app.Command("run-api", "Run server")
)

func main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case runClient.FullCommand():
		proxyServer := connection.RunProxyServer()
		connectionData, err := server.ConnectToHubAsClient(*api, "test", proxyServer)
		if err != nil {
			log.Panic(err)
		}

		err = web.Run(proxyServer, connectionData)
		if err != nil {
			log.Panic(err)
		}

	case runServer.FullCommand():
		proxyServer := connection.RunProxyServer()

		hub, err := server.ConnectToHubAsServer(*api, "test", proxyServer)
		if err != nil {
			log.Panic(err)
		}

		proxyOnClose := proxyServer.OnClose()
		hubOnClose := hub.OnClose()
		for {
			select {
			case <-hubOnClose:
				log.Println("Err: closed hub connection")
				os.Exit(1)
			case <-proxyOnClose:
				log.Println("Err: closed web server")
				os.Exit(1)
			case data := <-hub.OnConnection():
				log.Printf("Client %s is trying to connect, with id %d and secret: %s", data.IPs, data.ClientID, data.Secret)
				handleClient(proxyServer, data)
			}
		}
	}
}

func handleClient(proxyServer connection.Connection, data hub.OnConnectionData) {
	clientConnection, err := server.GetClientConnection(server.Secret(data.Secret), data.IPs)
	if err != nil {
		log.Println(err)
	}

	binaryClientId := make([]byte, 8)
	binary.LittleEndian.PutUint64(binaryClientId, data.ClientID)

	proxyServer.OnData(data.ClientID, clientConnection.HandleData)
	clientConnection.OnSend(func(data []byte, ip *net.UDPAddr) error {
		return proxyServer.Send(data, ip, binaryClientId)
	})

	defer func() {
		recoverErr := recover()
		if recoverErr != nil {
			log.Println("Closed clientConnection: ", recoverErr)
			clientConnection.Stop()
			proxyServer.Clear(data.ClientID)
		}
		clientConnection = nil
	}()

	closed := make(chan string)
	clientConnection.StartServer(closed)

	go func() {
		c := <-closed
		log.Printf("Closed clientConnection with: %d, reason: %s", data.ClientID, c)
	}()
}
