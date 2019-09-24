package socks5

import (
	"net"
)

type remoteTunnel struct {
	tunnel *tunnel
}

// OpenRemoteTunnel open a remote tunnel that connect local to remote
func OpenRemoteTunnel(conn net.Conn) {
	self := new(remoteTunnel)
	self.tunnel = new(tunnel)
	self.tunnel.clientSock = CreateSock(
		conn,
		func() {
			self.onClientReadable()
		},
		func() {
			self.tunnel.onClientClosed()
		},
	)
	self.tunnel.clientSock.start()
}

func (remoteTunnel *remoteTunnel) handleConnected() {

}

func (remoteTunnel *remoteTunnel) onClientReadable() {
}

func (remoteTunnel *remoteTunnel) onRemoteReadable() {
}
