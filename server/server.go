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

func ConnectToHubAsServer(api, name string, proxy ProxyServer) (Hub, error) {

    hub := hub{
    	proxy: proxy,
	}

	err := hub.StartAsServer(api, name)
	if err != nil {
		return nil, err
	}

	return &hub, nil
}

func ConnectToHubAsClient(api, name string, proxy ProxyServer) (*OnConnectionData, error) {

	hub := hub{
		proxy: proxy,
	}

	data, err := hub.StartAsClient(api, name)
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
