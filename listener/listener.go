package listener

import (
	"fmt"
	"net"
	"os"
	"time"
)

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}

	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(1 * time.Minute)
	return tc, nil
}

func NewListener(net string, laddr string) (net.Listener, error) {
	switch net {
	case "unix":
		return newUnixListener(laddr)
	case "tcp", "":
		return newTCPListener(laddr)
	default:
		return nil, fmt.Errorf("unknown address type %s", net)
	}
}

func newUnixListener(laddr string) (net.Listener, error) {
	fi, err := os.Stat(laddr)
	if err == nil {
		if !isSocket(fi.Mode()) {
			return nil, fmt.Errorf("file %s exists and is not a socket", laddr)
		}

		if err := os.Remove(laddr); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	return net.Listen("unix", laddr)
}

func isSocket(m os.FileMode) bool {
	return m&os.ModeSocket != 0
}

func newTCPListener(laddr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", laddr)
	if err != nil {
		return nil, err
	}

	// TODO: add TLS support

	return tcpKeepAliveListener{ln.(*net.TCPListener)}, nil
}
