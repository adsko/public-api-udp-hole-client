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
	Server sends PING message for all client addresses with address
	Client sends PONG message for all server addresses with 0.0.0.0 address
	Server retries PING messages up to N retires with delay M
	Client receives server PONG message and selects first
	Client sends PONG message with two addresses: selected address and address which used to send
	Server receives messages and selects first

	ClientConnection is established, switch to connection module
*/

var (
	app           = kingpin.New("hole-app", "")
	rendezvousAPI = app.Flag("rendezvous-api", "Address of the API server").Required().String()
	secure        = app.Flag("secure", "Usage of secure rendezvousAPI").Bool()

	runClient = app.Command("run-client", "Run client")
	runServer = app.Command("run-api", "Run server")

	serverURL = runServer.Flag("server-url", "Address of source server").Required().String()
)

func main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case runClient.FullCommand():
		proxyServer := connection.RunProxyServer()
		connectionData, err := hub.StartAsClient(proxyServer, *rendezvousAPI, "test", *secure)
		if err != nil {
			log.Panic(err)
		}

		err = web.Run(proxyServer, connectionData)
		if err != nil {
			log.Panic(err)
		}

	case runServer.FullCommand():
		proxyServer := connection.RunProxyServer()

		hub, err := connectToHubAsServer(*rendezvousAPI, "test", proxyServer)
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
				handleClient(*serverURL, proxyServer, data)
			}
		}
	}
}

func handleClient(serverURL string, proxyServer connection.Connection, hubData hub.OnConnectionData) {
	clientConnection, err := server.GetClientConnection(server.Secret(hubData.ClientSecret), server.Secret(hubData.ServerSecret), hubData.IPs)
	if err != nil {
		log.Println(err)
	}

	binaryClientId := make([]byte, 8)
	binary.LittleEndian.PutUint64(binaryClientId, hubData.ClientID)

	proxyServer.OnData(hubData.ClientID, clientConnection.HandleData)
	clientConnection.OnSend(func(data []byte, ip *net.UDPAddr) error {
		return proxyServer.Send(data, ip, binaryClientId)
	})

	defer func() {
		recoverErr := recover()
		if recoverErr != nil {
			log.Println("Closed clientConnection: ", recoverErr)
			clientConnection.Stop()
			proxyServer.Clear(hubData.ClientID)
		}
		clientConnection = nil
	}()

	closed := make(chan string)
	clientConnection.StartServer(serverURL, closed)

	go func() {
		c := <-closed
		log.Printf("Closed clientConnection with: %d, reason: %s", hubData.ClientID, c)
	}()
}

func connectToHubAsServer(rendezvousAPI, name string, proxy connection.Connection) (hub.ServerHUB, error) {
	hubConnection, err := hub.StartAsServer(proxy, rendezvousAPI, name, *secure)
	if err != nil {
		return nil, err
	}

	return hubConnection, nil
}
