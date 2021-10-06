Serer/Client for UDP hole punching connection


### Requirements
 * Go 1.16+
 * API server (https://github.com/adsko/public-api-udp-hole-server) which is available for server and client.

### Build
 * `go build cmd/main.go`

### Usage
`<executable> --api=API [<flags>] <command> [<args> ...]`

Flags:                                                                           
 * `--help`     Show context-sensitive help (also try --help-long and --help-man).  
 * `--api=API`  Address of the API server (required)


Commands:                                                                        
 * `help [<command>...]` Show help.
 * `run-client` Run client
 * `run-api --server-url=SERVER-URL` Run server                                                                   


### Example usage:
 * `main.exe --api=make-public-server.herokuapp.com:443 run-api --server-url=localhost:4200`
 * `main.exe --api=make-public-server.herokuapp.com:443 run-client`