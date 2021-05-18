package server

type register struct {
	Name string
	Port string
	Addr []string
	Password string
}

type openConnection struct {
	Addr []string
	Port string
	ClientID uint64
	Secret string
}

func ConnectToHubAsServer(name string, proxy ProxyServer) (Hub, error) {

	addr := "localhost:10000"
    hub := hub{
    	proxy: proxy,
	}

	err := hub.StartAsServer(addr, name)
	if err != nil {
		return nil, err
	}

	return &hub, nil
}

func ConnectToHubAsClient(name string, proxy ProxyServer) (*OnConnectionData, error) {

	addr := "localhost:10000"
	hub := hub{
		proxy: proxy,
	}

	data, err := hub.StartAsClient(addr, name)
	if err != nil {
		return nil, err
	}

	return data, nil
}


func RunProxyServer() ProxyServer {
	proxy := proxyServer{
	}
	proxy.Start()

	return &proxy
}
