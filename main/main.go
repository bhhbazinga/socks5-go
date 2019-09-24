package main

import (
	"log"
	"net"
	"socks5"
)

func localStart() {
	remote, err := net.Dial("tcp", "192.168.1.40")
	if err != nil {
		log.Fatal(err)
	}
	socks5.CreateSock(
		remote,
		func() {

		},
		func() {

		},
	)

	local, err := net.Listen("tcp", ":8022")
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := local.Accept()
		if err != nil {
			log.Println("accept err:", err)
		} else {
			socks5.OpenLocalTunnel(conn, remote)
		}
	}
}

func remoteStart() {

}

func main() {
	localStart()
}
