# socks5
A light weight socks5 proxy server implemented in go which contains a local side and a remote side.
you can use it to surf the Internet scientifically!
## Suport
- AES encryption
- Ipv4 & Ipv6
- "CONNECT" command
## Run
```
git clone https://github.com/bhhbazinga/socks5-go.git
mv socks5-go $GOPATH/src
cd $GOPATH/src/socks5-go/main
go run main.go
```
## Usage
```
Usage:
  -k string
        a 16bytes key of AES128 (default "1234567890qwerty")
  -l    run as local server
  -la string
        local ip address
  -lp string
        local port
  -r    run as remote server
  -ra string
        remote ip address
  -rp string
        remote port

Examples:
Run as local:-l -la 0.0.0.0 -lp 8001 -ra 10.101.200.20 -rp 8002 -k 1234567890qwerty
Run as remote:-r -ra 0.0.0.0 -rp 8002 -k 1234567890qwerty

Finaly, if you use chrome, you can install the SwitchyOmega plug-in, and configure the local startup parameters.
