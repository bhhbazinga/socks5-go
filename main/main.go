package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"socks5-go"
)

var (
	l bool
	r bool

	la string
	lp string

	ra string
	rp string

	k string
)

func localStart() {
	local, err := net.Listen("tcp", fmt.Sprintf("%s:%s", la, lp))
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := local.Accept()
		if err != nil {
			log.Println("accept err:", err)
		} else {
			socks5.OpenLocalTunnel(conn, fmt.Sprintf("%s:%s", ra, rp), k)
		}
	}
}

func remoteStart() {
	remote, err := net.Listen("tcp", fmt.Sprintf("%s:%s", ra, rp))
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := remote.Accept()
		if err != nil {
			log.Println("accept err:", err)
		} else {
			socks5.OpenRemoteTunnel(conn, k)
		}
	}
}

func usage() {
	fmt.Fprint(os.Stderr, "Usage:\n")
	flag.PrintDefaults()
	fmt.Fprint(os.Stderr, "\n")
	fmt.Fprint(os.Stderr, "Examples:\n")
	fmt.Fprint(os.Stderr, "Run as local:")
	fmt.Fprint(os.Stderr, "-l -la 0.0.0.0 -lp 8001 -ra 10.101.200.20 -rp 8002 -k 1234567890qwerty\n")
	fmt.Fprint(os.Stderr, "Run as remote:")
	fmt.Fprint(os.Stderr, "-r -ra 0.0.0.0 -rp 8002 -k 1234567890qwerty\n")
	os.Exit(1)
}

func init() {
	flag.BoolVar(&l, "l", false, "run as local server")
	flag.BoolVar(&r, "r", false, "run as remote server")

	flag.StringVar(&la, "la", "", "local ip address")
	flag.StringVar(&lp, "lp", "", "local port")

	flag.StringVar(&ra, "ra", "", "remote ip address")
	flag.StringVar(&rp, "rp", "", "remote port")

	flag.StringVar(&k, "k", "1234567890qwerty", "a 16bytes key of AES128")
	flag.Usage = usage
}

func main() {
	flag.Parse()
	if l == false && r == false {
		usage()
	}

	if l && r {
		fmt.Fprintln(os.Stderr, "choose only -l or -r")
		os.Exit(1)
	}

	if ra == "" || rp == "" {
		usage()
	}

	if l {
		if la == "" || lp == "" {
			usage()
		}
		localStart()
	} else {
		remoteStart()
	}
}
