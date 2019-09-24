package socks5

// import (
// 	"bytes"
// 	"encoding/binary"
// 	"fmt"
// 	"log"
// 	"net"
// )

// type localTunnelState int

// const (
// 	open localTunnelState = iota
// 	request
// 	connecting
// 	connected
// )

// type openProtocol struct {
// 	ver      byte
// 	nmethods byte
// 	methods  []byte
// }

// type requestProtocol struct {
// 	ver       byte
// 	cmd       byte
// 	rsv       byte
// 	atyp      byte
// 	domainlen uint8
// 	addr      []byte
// 	port      uint16
// }

// type connectedProtocol struct {
// 	len  uint8
// 	atyp byte
// 	addr []byte
// 	port uint16
// }

// type localTunnel struct {
// 	tunnel      *tunnel
// 	op          *openProtocol
// 	rp          *requestProtocol
// 	cp          *connectedProtocol
// 	state       localTunnelState
// 	readedCount int
// }

// // OpenLocalTunnel open a local tunnel that connect browser to local
// func OpenLocalTunnel(conn net.Conn, connRemote net.Conn) {
// 	localTunnel := new(localTunnel)
// 	localTunnel.state = open
// 	localTunnel.readedCount = 0
// 	localTunnel.op = new(openProtocol)
// 	localTunnel.rp = new(requestProtocol)
// 	localTunnel.tunnel = new(tunnel)
// 	localTunnel.tunnel.clientSock = CreateSock(
// 		conn,
// 		func() {
// 			localTunnel.onClientReadable()
// 		},
// 		func() {
// 			localTunnel.tunnel.onClientClosed()
// 		},
// 	)
// 	localTunnel.tunnel.clientSock.start()

// 	localTunnel.tunnel.remoteSock = CreateSock(
// 		conn,
// 		func() {
// 			localTunnel.onRemoteReadable()
// 		},
// 		func() {
// 			log.Fatal("remote server crashed or key is invalid")
// 		},
// 	)
// 	localTunnel.tunnel.remoteSock.start()
// 	/*
// 		request connect to remote:
// 		|cmd(byte)|key(var []byte)|
// 	*/
// 	key := "123456"
// 	localTunnel.tunnel.remoteSock.write([]byte{0x01})
// 	localTunnel.tunnel.remoteSock.write([]byte(key))
// }

// func (localTunnel *localTunnel) openHandle() {
// 	var buff []byte
// 	clientSock := localTunnel.tunnel.clientSock
// 	readBuff := clientSock.readBuff
// 	op := localTunnel.op

// 	if localTunnel.readedCount == 0 {
// 		goto header
// 	} else if localTunnel.readedCount == 2 {
// 		goto methods
// 	} else {
// 		log.Fatal("openHandle error")
// 	}

// header:
// 	if readBuff.Len() < 2 {
// 		return
// 	}
// 	buff = make([]byte, 2)
// 	readBuff.Read(buff)
// 	op.ver, op.nmethods = buff[0], buff[1]
// 	if op.ver != 0x05 {
// 		clientSock.shutdown()
// 		return
// 	}

// methods:
// 	if readBuff.Len() < int(localTunnel.op.nmethods) {
// 		return
// 	}
// 	buff = make([]byte, int(localTunnel.op.nmethods))
// 	readBuff.Read(buff)
// 	op.methods = buff

// 	localTunnel.readedCount = 0
// 	localTunnel.state = request
// 	localTunnel.tunnel.writeClient([]byte{byte(0x05), byte(0x00)})
// }

// func (localTunnel *localTunnel) parseIP(readBuff *bytes.Buffer, atyp byte) *net.TCPAddr {
// 	var len int
// 	rp := localTunnel.rp
// 	if atyp == 0x01 {
// 		len = 6
// 	} else {
// 		len = 18
// 	}

// 	if readBuff.Len() < len {
// 		return nil
// 	}

// 	buff := make([]byte, len)
// 	readBuff.Read(buff)

// 	tcpAddr := new(net.TCPAddr)
// 	rp.addr, rp.port = buff[:len-2], binary.BigEndian.Uint16(buff[len-2:])
// 	tcpAddr.IP, tcpAddr.Port = rp.addr, int(rp.port)
// 	return tcpAddr
// }

// func (localTunnel *localTunnel) parseDomain(readBuff *bytes.Buffer) (*net.TCPAddr, error) {
// 	var buff []byte
// 	rp := localTunnel.rp
// 	if localTunnel.readedCount == 4 {
// 		if readBuff.Len() < 1 {
// 			return nil, nil
// 		}
// 		localTunnel.rp.domainlen, _ = readBuff.ReadByte()
// 		localTunnel.readedCount++
// 	}
// 	// parse domain
// 	if readBuff.Len() < int(localTunnel.rp.domainlen) {
// 		return nil, nil
// 	}
// 	buff = make([]byte, rp.domainlen)
// 	readBuff.Read(buff)
// 	rp.addr = buff
// 	buff = make([]byte, 2)
// 	readBuff.Read(buff)
// 	rp.port = binary.BigEndian.Uint16(buff)
// 	host := fmt.Sprintf("%s:%d", string(rp.addr), rp.port)
// 	return net.ResolveTCPAddr("tcp", host)
// }

// func (localTunnel *localTunnel) connectToRemote(tcpAddr *net.TCPAddr) (*Sock, error) {
// 	conn, err := net.DialTCP("tcp", nil, tcpAddr)
// 	if err != nil {
// 		return nil, err
// 	}
// 	remoteSock := CreateSock(
// 		conn,
// 		func() {
// 			localTunnel.onRemoteReadable()
// 		},
// 		func() {
// 			localTunnel.tunnel.onRemoteClosed()
// 		},
// 	)
// 	localTunnel.tunnel.remoteSock = remoteSock
// 	return remoteSock, nil
// }

// func (localTunnel *localTunnel) requestHandle() {
// 	var buff []byte
// 	clientSock := localTunnel.tunnel.clientSock
// 	readBuff := clientSock.readBuff
// 	rp := localTunnel.rp
// 	if localTunnel.readedCount == 0 {
// 		goto header
// 	} else if localTunnel.readedCount == 4 {
// 		goto addr
// 	} else {
// 		log.Fatal("requestHandle error")
// 	}

// header:
// 	if readBuff.Len() < 4 {
// 		return
// 	}
// 	buff = make([]byte, 4)
// 	readBuff.Read(buff)
// 	rp.ver, rp.cmd, rp.rsv, rp.atyp = buff[0], buff[1], buff[2], buff[3]
// 	localTunnel.readedCount = 4
// addr:
// 	var tcpAddr *net.TCPAddr
// 	var err error
// 	switch rp.atyp {
// 	case 0x01: // ipv4
// 		fallthrough
// 	case 0x04: // ipv6
// 		tcpAddr = localTunnel.parseIP(readBuff, rp.atyp)
// 		if tcpAddr == nil {
// 			return
// 		}
// 	case 0x03: // domain
// 		// parse domainlen
// 		tcpAddr, err = localTunnel.parseDomain(readBuff)
// 		if err != nil {
// 			log.Println("domain resolve failed")
// 			clientSock.shutdown()
// 			return
// 		}
// 	default:
// 		log.Println("unexpected atyp")
// 		clientSock.shutdown()
// 		return
// 	}

// 	localTunnel.readedCount = 0
// 	localTunnel.state = connecting
// 	/*
// 		|cmd(byte)|len(uint8)|encrypt(|atype|ip|port|)|
// 		cmd:
// 		0x01:connect
// 	*/

// 	// TODO
// 	replyBuffer := new(bytes.Buffer)
// 	replyBuffer.Write()

// 	localTunnel.writeRemote([]byte{0x01})

// 	// tcpAddr.IP
// 	// remoteSock, err := localTunnel.connectToRemote(tcpAddr)
// 	// if err != nil {
// 	// 	log.Println("connectToRemote err:", err)
// 	// 	clientSock.shutdown()
// 	// 	return
// 	// }

// 	// localIPPortStr := strings.Split(remoteSock.conn.LocalAddr().String(), ":")
// 	// localIPStr, localPortStr := localIPPortStr[0], localIPPortStr[1]
// 	// localIP := net.ParseIP(localIPStr)
// 	// localPort, _ := strconv.Atoi(localPortStr)
// 	// ipv4Buff := localIP.To4()
// 	// localTunnel.tunnel.writeClient([]byte{0x05, 0x00, 0x00, 0x01})
// 	// replyBuffer := new(bytes.Buffer)
// 	// if ipv4Buff != nil {
// 	// 	replyBuffer.Write(ipv4Buff)
// 	// } else {
// 	// 	ipv6Buff := localIP.To16()
// 	// 	replyBuffer.Write(ipv6Buff)
// 	// }
// 	// binary.Write(replyBuffer, binary.BigEndian, uint16(localPort))
// 	// localTunnel.tunnel.writeClient(replyBuffer.Bytes())
// }

// func (localTunnel *localTunnel) forwardClientHandle() {
// 	localTunnel.tunnel.forward(localTunnel.tunnel.clientSock, localTunnel.tunnel.remoteSock)
// }

// func (localTunnel *localTunnel) onClientReadable() {
// 	switch localTunnel.state {
// 	case open:
// 		localTunnel.openHandle()
// 	case request:
// 		localTunnel.requestHandle()
// 	case connected:
// 		localTunnel.forwardClientHandle()
// 	}
// }

// // |len(uint8)|encrypt(|atype|ip|port|)|
// func (localTunnel *localTunnel) connectingHandle() {
// 	var buff []byte
// 	remoteSock := localTunnel.tunnel.remoteSock
// 	readBuff := remoteSock.readBuff
// 	cp := localTunnel.cp
// 	if localTunnel.readedCount == 0 {
// 		goto header
// 	} else if localTunnel.readedCount == 1 {
// 		goto data
// 	} else {
// 		log.Fatal("connectingHandle error")
// 	}

// header:
// 	if readBuff.Len() < 1 {
// 		return
// 	}
// 	cp.len = readBuff.ReadByte()
// 	localTunnel.readedCount = 1
// data:
// 	if readBuff.Len() < cp.len {
// 		return
// 	}
// }

// func (localTunnel *localTunnel) forwardRemoteHandle() {

// }

// // |len(uint8)|encrypt(var []byte)|
// func (localTunnel *localTunnel) onRemoteReadable() {
// 	readBuff := localTunnel.tunnel.remoteSock.readBuff
// 	switch localTunnel.state {
// 	case connecting:
// 		localTunnel.connectingHandle()
// 	case connected:
// 		localTunnel.forwardRemoteHandle()
// 	}
// 	// localTunnel.tunnel.forward(localTunnel.tunnel.remoteSock, localTunnel.tunnel.clientSock)
// }

// // type localServerState int

// // const (
// // 	localAuth localServerState = iota
// // 	localConnected
// // )

// // // LocalServer is a socket that connect local to remote, we should initiate it when local start
// // type LocalServer struct {
// // 	sock      *Sock
// // 	state     localServerState
// // 	tunnelMap map[int]*localTunnel
// // }

// // // CreateLocalServer define
// // func CreateLocalServer(conn net.Conn) *LocalServer {
// // 	self := new(LocalServer)
// // 	self.sock = CreateSock(
// // 		conn,
// // 		func() {
// // 			self.onRemoteReadable()
// // 		},
// // 		func() {
// // 			self.onRemoteReadable()
// // 		},
// // 	)
// // 	go self.sock.start()

// // 	/*
// // 		request connect to remote:
// // 		|cmd(byte)|key(var []byte)|
// // 	*/

// // 	key := "123456"
// // 	self.sock.write([]byte{0x01})
// // 	self.sock.write([]byte(key))
// // 	return self
// // }

// // /*
// // |cmd(byte)|tunnelId(int)|datalen(uint32)|data(var []byte)|
// // cmd:
// // 0x00:connected
// // 0x01:forward data
// // 0x02:close tunnel
// // */
// // func (localServer *LocalServer) onRemoteReadable() {
// // 	readBuff := localServer.sock.readBuff
// // 	if readBuff.Len() < 5 {
// // 		return
// // 	}
// // 	cmd := readBuff.ReadByte()
// // 	cmd := readBuff.r
// // }

// // func (localServer *LocalServer) onRemoteClosed() {
// // 	log.Fatal("connect to remote failed, key is invalid or remote crashed")
// // }
