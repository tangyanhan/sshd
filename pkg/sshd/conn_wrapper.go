package sshd

import (
	"net"

	"github.com/gliderlabs/ssh"
)

type sshConn struct {
	net.Conn
	closeCallback func(string)
	ctx           ssh.Context
}

func (c *sshConn) Close() error {
	if id, ok := c.ctx.Value(ssh.ContextKeySessionID).(string); ok {
		c.closeCallback(id)
	}

	return c.Conn.Close()
}
