Serer/Client for UDP hole punching connection.

For example echo server, see `examples` directory.


### Requirements
 * Go 1.16+
 * Rendezvous API server (https://github.com/adsko/rendezvous-udp-hole-server) which is available for server and client.

### Build
 * `go build cmd/main.go`

### Usage
`<executable> --api=API [<flags>] <command> [<args> ...]`

Flags:                                                                           
 * `--help`     Show context-sensitive help (also try --help-long and --help-man).  
 * `--rendezvous-api=API`  Address of the rendezvous API server which is required for connection establishment (required)
 * `--secure`   Use secure schemes in connection (https/wss)


Commands:                                                                        
 * `help [<command>...]` Show help.
 * `run-client` Run client
 * `run-api --server-url=SERVER-URL` Run server, where server url is destination server.                                                              


### Example usage:
 * `main.exe --rendezvous-api=make-public-server.herokuapp.com:443 --secure run-api --server-url=localhost:4200`
 * `main.exe --rendezvous-api=make-public-server.herokuapp.com:443 --secure run-client`