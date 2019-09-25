package socks5

import (
	"bytes"
	"io"
	"log"
	"net"
)

var key = "1234567890qwerty"
var buffSize = 1024

const chanSize = 1024

// ReadableFunc as a callback when socket readable or closed
type ReadableFunc func()

// Sock define a socket
type Sock struct {
	conn             net.Conn
	readBuff         *bytes.Buffer
	writeChan        chan []byte
	readableCallback ReadableFunc
	closedCallback   ReadableFunc
	closed           bool
}

type tunnel struct {
	clientSock *Sock
	remoteSock *Sock
}

func (tunnel *tunnel) forward(fromSock *Sock, toSock *Sock) {
	if fromSock == nil || toSock == nil {
		return
	}

	readBuff := fromSock.readBuff
	buff := make([]byte, readBuff.Len())
	readBuff.Read(buff)
	tunnel.writeSock(toSock, buff)
}

func (tunnel *tunnel) onClientClosed() {
	tunnel.clientSock = nil
}

func (tunnel *tunnel) onRemoteClosed() {
	tunnel.remoteSock = nil
}

func (tunnel *tunnel) writeClient(buff []byte) {
	tunnel.writeSock(tunnel.clientSock, buff)
}

func (tunnel *tunnel) writeRemote(buff []byte) {
	tunnel.writeSock(tunnel.remoteSock, buff)
}

func (tunnel *tunnel) writeSock(sock *Sock, buff []byte) {
	if sock != nil {
		sock.write(buff)
	}
}

func (tunnel *tunnel) shutdown() {
	if tunnel.clientSock != nil {
		tunnel.clientSock.shutdown()
	}
	if tunnel.remoteSock != nil {
		tunnel.remoteSock.shutdown()
	}
}

// CreateSock : create a socket
func CreateSock(conn net.Conn, readableCb ReadableFunc, closedCb ReadableFunc) *Sock {
	sock := new(Sock)
	sock.conn = conn
	sock.readBuff = new(bytes.Buffer)
	sock.readableCallback = readableCb
	sock.closedCallback = closedCb
	sock.writeChan = make(chan []byte, chanSize)
	sock.closed = false
	return sock
}

func (sock *Sock) start() {
	go func() {
		for {
			if sock.closed {
				return
			}

			buff := make([]byte, buffSize)
			n, err := sock.conn.Read(buff)
			if n >= buffSize {
				buffSize *= 2
			}
			if err != nil {
				if err != io.EOF {
					log.Println("sock read err:", err)
				}
				sock.shutdown()
				return
			}
			sock.readBuff.Write(buff[:n])
			sock.readableCallback()
		}
	}()

	go func() {
		for {
			if sock.closed {
				return
			}

			buff := <-sock.writeChan
			_, err := sock.conn.Write(buff)
			if err != nil {
				log.Println("sock write err:", err)
				sock.shutdown()
				return
			}
		}
	}()
}

func (sock *Sock) write(buff []byte) {
	sock.writeChan <- buff
}

func (sock *Sock) shutdown() {
	if sock.closed {
		return
	}
	sock.closed = true
	sock.conn.Close()
	sock.closedCallback()
}
