package socks5

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

type protocol struct {
	len uint32
	// data []byte
}

type remoteTunnel struct {
	tunnel      *tunnel
	state       localTunnelState
	p           *protocol
	readedCount int
}

// OpenRemoteTunnel open a remote tunnel that connect local to remote
func OpenRemoteTunnel(conn net.Conn) {
	remoteTunnel := new(remoteTunnel)
	remoteTunnel.tunnel = new(tunnel)
	remoteTunnel.tunnel.clientSock = CreateSock(
		conn,
		func() {
			remoteTunnel.onClientReadable()
		},
		func() {
			remoteTunnel.tunnel.onClientClosed()
		},
	)
	remoteTunnel.tunnel.clientSock.start()
	remoteTunnel.p = new(protocol)
	remoteTunnel.readedCount = 0
	remoteTunnel.state = request
}

func (remoteTunnel *remoteTunnel) parseIP(data []byte) *net.TCPAddr {
	var len int
	if data[0] == 0x01 {
		len = 6
	} else {
		len = 18
	}
	tcpAddr := new(net.TCPAddr)
	tcpAddr.IP, tcpAddr.Port = data[1:len-2], int(binary.BigEndian.Uint16(data[len-2:]))
	return tcpAddr
}

func (remoteTunnel *remoteTunnel) parseDomain(data []byte) (*net.TCPAddr, error) {
	len := len(data)
	domain := data[1 : len-2]
	port := binary.BigEndian.Uint16(data[len-2:])
	host := fmt.Sprintf("%s:%d", string(domain), port)
	return net.ResolveTCPAddr("tcp", host)
}

func (remoteTunnel *remoteTunnel) connectToRemote(tcpAddr *net.TCPAddr) (*Sock, error) {
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}

	remoteSock := CreateSock(
		conn,
		func() {
			remoteTunnel.onRemoteReadable()
		},
		func() {
			remoteTunnel.tunnel.onRemoteClosed()
		},
	)
	remoteTunnel.tunnel.remoteSock = remoteSock
	return remoteSock, nil
}

// |len(uint8)|encrypt(|atype|addr|port|)|
func (remoteTunnel *remoteTunnel) requestHandle() {
	var buff []byte
	readBuff := remoteTunnel.tunnel.clientSock.readBuff
	p := remoteTunnel.p
	if remoteTunnel.readedCount == 0 {
		goto header
	} else if remoteTunnel.readedCount == 4 {
		goto data
	} else {
		log.Fatal("requestHandle err")
	}

header:
	if readBuff.Len() < 1 {
		return
	}

	buff = make([]byte, 4)
	readBuff.Read(buff)
	p.len = binary.BigEndian.Uint32(buff)
	remoteTunnel.readedCount = 4

data:
	if readBuff.Len() < int(p.len) {
		return
	}
	buff = make([]byte, p.len)
	readBuff.Read(buff)
	decryptData, err := aesDecrypt(buff, []byte(key))
	if err != nil {
		log.Println("aesDecrypt err:", err)
		remoteTunnel.tunnel.shutdown()
		return
	}

	var tcpAddr *net.TCPAddr
	switch decryptData[0] {
	case 0x01: // ipv4
		fallthrough
	case 0x04: // ipv6
		tcpAddr = remoteTunnel.parseIP(decryptData)
		if tcpAddr == nil {
			return
		}
	case 0x03: // domain
		tcpAddr, err = remoteTunnel.parseDomain(decryptData)
		if err != nil {
			log.Println("domain resolve failed:", err)
			remoteTunnel.tunnel.shutdown()
			return
		}
	default:
		log.Println("unexpected atyp")
		remoteTunnel.tunnel.shutdown()
		return
	}

	remoteSock, err := remoteTunnel.connectToRemote(tcpAddr)
	if err != nil {
		log.Println("connectToRemote err:", err)
		remoteTunnel.tunnel.shutdown()
		return
	}

	localIPPortStr := strings.Split(remoteSock.conn.LocalAddr().String(), ":")
	localIPStr, localPortStr := localIPPortStr[0], localIPPortStr[1]
	localIP := net.ParseIP(localIPStr)
	localPort, _ := strconv.Atoi(localPortStr)
	ipv4Buff := localIP.To4()
	replyBuffer := new(bytes.Buffer)
	if ipv4Buff != nil {
		replyBuffer.WriteByte(0x01)
		replyBuffer.Write(ipv4Buff)
	} else {
		replyBuffer.WriteByte(0x04)
		ipv6Buff := localIP.To16()
		replyBuffer.Write(ipv6Buff)
	}
	binary.Write(replyBuffer, binary.BigEndian, uint16(localPort))

	encryptData, err := aesEncrypt(replyBuffer.Bytes(), []byte(key))
	if err != nil {
		log.Println("aesEncrypt err:", err)
		remoteTunnel.tunnel.shutdown()
		return
	}

	remoteTunnel.tunnel.writeClient([]byte{uint8(len(encryptData))})
	remoteTunnel.tunnel.writeClient(encryptData)

	remoteTunnel.readedCount = 0
	remoteTunnel.state = connected
	remoteSock.start()
}

// |len(uint32)|encrypt(data []byte)|
func (remoteTunnel *remoteTunnel) forwardClientHandle() {
start:
	readBuff := remoteTunnel.tunnel.clientSock.readBuff
	p := remoteTunnel.p
	var buff []byte
	if remoteTunnel.readedCount == 0 {
		goto header
	} else if remoteTunnel.readedCount == 4 {
		goto data
	} else {
		log.Fatal("forwardClientHandle err")
	}

header:
	if readBuff.Len() < 4 {
		return
	}

	buff = make([]byte, 4)
	readBuff.Read(buff)
	p.len = binary.BigEndian.Uint32(buff)
	remoteTunnel.readedCount = 4

data:
	if readBuff.Len() < int(p.len) {
		return
	}
	buff = make([]byte, p.len)
	readBuff.Read(buff)
	decryptData, err := aesDecrypt(buff, []byte(key))
	if err != nil {
		log.Println("aesDecrypt err:", err)
		remoteTunnel.tunnel.shutdown()
		return
	}

	remoteTunnel.tunnel.writeRemote(decryptData)
	remoteTunnel.readedCount = 0
	goto start
}

func (remoteTunnel *remoteTunnel) onClientReadable() {
	switch remoteTunnel.state {
	case request:
		remoteTunnel.requestHandle()
	case connected:
		remoteTunnel.forwardClientHandle()
	}
}

func (remoteTunnel *remoteTunnel) onRemoteReadable() {
	if remoteTunnel.state != connected {
		log.Fatal("onRemoteReadable remoteTunnel.state != connected")
	}

	readBuff := remoteTunnel.tunnel.remoteSock.readBuff
	encryptData, err := aesEncrypt(readBuff.Bytes(), []byte(key))
	if err != nil {
		log.Println("aesEncrypt err:", err)
		remoteTunnel.tunnel.shutdown()
	}
	buff := make([]byte, 4)
	binary.BigEndian.PutUint32(buff, uint32(len(encryptData)))
	remoteTunnel.tunnel.writeClient(buff)
	remoteTunnel.tunnel.writeClient(encryptData)
	readBuff.Reset()
}
