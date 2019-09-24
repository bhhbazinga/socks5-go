package socks5

import (
	"bytes"
	"log"
	"net"
)

type localTunnelState int

const (
	open localTunnelState = iota
	request
	connecting
	connected
)

type openProtocol struct {
	ver      byte
	nmethods byte
	methods  []byte
}

type requestProtocol struct {
	ver       byte
	cmd       byte
	rsv       byte
	atyp      byte
	domainlen uint8
	addr      []byte
	port      uint16
}

type connectedProtocol struct {
	len  uint8
	atyp byte
	addr []byte
	port uint16
}

type forwardProtocol struct {
	len  uint32
	data []byte
}

type localTunnel struct {
	tunnel      *tunnel
	op          *openProtocol
	rp          *requestProtocol
	cp          *connectedProtocol
	fp          *forwardProtocol
	state       localTunnelState
	readedCount int
}

const key = "123456"

// OpenLocalTunnel open a local tunnel that connect browser to local
func OpenLocalTunnel(conn net.Conn, connRemote net.Conn) (err error) {
	localTunnel := new(localTunnel)
	localTunnel.state = open
	localTunnel.readedCount = 0
	localTunnel.op = new(openProtocol)
	localTunnel.rp = new(requestProtocol)
	localTunnel.cp = new(connectedProtocol)
	localTunnel.fp = new(forwardProtocol)
	localTunnel.tunnel = new(tunnel)
	localTunnel.tunnel.clientSock = CreateSock(
		conn,
		func() {
			localTunnel.onClientReadable()
		},
		func() {
			localTunnel.tunnel.onClientClosed()
		},
	)
	localTunnel.tunnel.clientSock.start()

	remoteConn, err := net.Dial("tcp", "192.168.1.40:18822")
	if err != nil {
		localTunnel.tunnel.clientSock.shutdown()
		return err
	}
	localTunnel.tunnel.remoteSock = CreateSock(
		remoteConn,
		func() {
			localTunnel.onRemoteReadable()
		},
		func() {
			log.Fatal("remote server crashed or key is invalid")
		},
	)
	localTunnel.tunnel.remoteSock.start()
	/*
		request connect to remote:
		|cmd(byte)|key(var []byte)|
	*/

	localTunnel.tunnel.remoteSock.write([]byte{0x01})
	localTunnel.tunnel.remoteSock.write([]byte(key))
	return nil
}

func (localTunnel *localTunnel) openHandle() {
	var buff []byte
	clientSock := localTunnel.tunnel.clientSock
	readBuff := clientSock.readBuff
	op := localTunnel.op

	if localTunnel.readedCount == 0 {
		goto header
	} else if localTunnel.readedCount == 2 {
		goto methods
	} else {
		log.Fatal("openHandle error")
	}

header:
	if readBuff.Len() < 2 {
		return
	}
	buff = make([]byte, 2)
	readBuff.Read(buff)
	op.ver, op.nmethods = buff[0], buff[1]
	if op.ver != 0x05 {
		localTunnel.tunnel.shutdown()
		return
	}

methods:
	if readBuff.Len() < int(localTunnel.op.nmethods) {
		return
	}
	buff = make([]byte, int(localTunnel.op.nmethods))
	readBuff.Read(buff)
	op.methods = buff

	localTunnel.readedCount = 0
	localTunnel.state = request
	localTunnel.tunnel.writeClient([]byte{byte(0x05), byte(0x00)})
}

func (localTunnel *localTunnel) requestHandle() {
	var buff []byte
	clientSock := localTunnel.tunnel.clientSock
	readBuff := clientSock.readBuff
	rp := localTunnel.rp
	if localTunnel.readedCount == 0 {
		goto header
	} else if localTunnel.readedCount == 4 {
		goto addr
	} else {
		log.Fatal("requestHandle error")
	}

header:
	if readBuff.Len() < 4 {
		return
	}
	buff = make([]byte, 4)
	readBuff.Read(buff)
	rp.ver, rp.cmd, rp.rsv, rp.atyp = buff[0], buff[1], buff[2], buff[3]
	localTunnel.readedCount = 4
addr:
	var replyBuffer *bytes.Buffer
	switch rp.atyp {
	case 0x01: // ipv4
		if readBuff.Len() < 6 {
			return
		}

		buff = make([]byte, readBuff.Len())
		readBuff.Read(buff)
		replyBuffer = new(bytes.Buffer)
		replyBuffer.WriteByte(rp.atyp)
		replyBuffer.Write(buff)
	case 0x04: // ipv6
		if readBuff.Len() < 18 {
			return
		}

		buff = make([]byte, readBuff.Len())
		readBuff.Read(buff)
		replyBuffer = new(bytes.Buffer)
		replyBuffer.WriteByte(rp.atyp)
		replyBuffer.Write(buff)
	case 0x03: // domain
		if localTunnel.readedCount == 4 {
			if readBuff.Len() < 1 {
				return
			}
			localTunnel.rp.domainlen, _ = readBuff.ReadByte()
			localTunnel.readedCount++
		}

		if readBuff.Len() < int(localTunnel.rp.domainlen) {
			return
		}
		buff = make([]byte, rp.domainlen)
		readBuff.Read(buff)
		replyBuffer = new(bytes.Buffer)
		replyBuffer.WriteByte(rp.atyp)
		replyBuffer.WriteByte(rp.domainlen)
		replyBuffer.Write(buff)
	default:
		log.Println("unexpected atyp")
		localTunnel.tunnel.shutdown()
		return
	}

	localTunnel.readedCount = 0
	localTunnel.state = connecting

	encryptData, err := aesEncrypt(replyBuffer.Bytes(), []byte(key))
	if err != nil {
		log.Println("aesEncrypt err:", err)
		localTunnel.tunnel.shutdown()
		return
	}
	/*
		|len(uint8)|encrypt(|atype|ip|port|)|
		cmd:
		0x01:connect
	*/
	localTunnel.tunnel.writeRemote([]byte{byte(len(encryptData))})
	localTunnel.tunnel.writeRemote(encryptData)
}

func (localTunnel *localTunnel) forwardClientHandle() {
	readBuff := localTunnel.tunnel.clientSock.readBuff
	encryptData, err := aesEncrypt(readBuff.Bytes(), []byte(key))
	if err != nil {
		log.Println("aesEncrypt err:", err)
		localTunnel.tunnel.shutdown()
		return
	}
	readBuff.Reset()
	localTunnel.tunnel.writeRemote([]byte{byte(len(encryptData))})
	localTunnel.tunnel.writeRemote(encryptData)
}

func (localTunnel *localTunnel) onClientReadable() {
	switch localTunnel.state {
	case open:
		localTunnel.openHandle()
	case request:
		localTunnel.requestHandle()
	case connected:
		localTunnel.forwardClientHandle()
	}
}

// |len(uint8)|encrypt(|atype|ip|port|)|
func (localTunnel *localTunnel) connectingHandle() {
	remoteSock := localTunnel.tunnel.remoteSock
	readBuff := remoteSock.readBuff
	cp := localTunnel.cp
	if localTunnel.readedCount == 0 {
		goto header
	} else if localTunnel.readedCount == 1 {
		goto data
	} else {
		log.Fatal("connectingHandle error")
	}

header:
	if readBuff.Len() < 1 {
		return
	}
	cp.len, _ = readBuff.ReadByte()
	localTunnel.readedCount = 1

data:
	if readBuff.Len() < int(cp.len) {
		return
	}
	buff := make([]byte, cp.len)
	readBuff.Read(buff)
	decryptData, err := aesDecrypt(buff, []byte(key))
	if err != nil {
		localTunnel.tunnel.shutdown()
		log.Println("aesDecrypt err:", err)
		return
	}
	localTunnel.state = connected
	localTunnel.tunnel.writeClient([]byte{0x05, 0x00, 0x00})
	localTunnel.tunnel.writeClient(decryptData)
}

// |len(uint32)|encrypt(data []byte)|
func (localTunnel *localTunnel) forwardRemoteHandle() {
	remoteSock := localTunnel.tunnel.remoteSock
	readBuff := remoteSock.readBuff
	fp := localTunnel.fp
	if localTunnel.readedCount == 0 {
		goto header
	} else if localTunnel.readedCount == 1 {
		goto data
	} else {
		log.Fatal("connectingHandle error")
	}

header:
	if readBuff.Len() < 1 {
		return
	}

data:
	if readBuff.Len() < int(fp.len) {
		return
	}
	buff := make([]byte, fp.len)
	readBuff.Read(buff)
	decryptData, err := aesDecrypt(buff, []byte(key))
	if err != nil {
		log.Println("aesDecrypt err:", err)
		return
	}
	localTunnel.tunnel.writeClient(decryptData)
}

func (localTunnel *localTunnel) onRemoteReadable() {
	switch localTunnel.state {
	case connecting:
		localTunnel.connectingHandle()
	case connected:
		localTunnel.forwardRemoteHandle()
	}
}
