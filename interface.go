// Package reuseport provides Listen and Dial functions that set socket options
// in order to be able to reuse ports. You should only use this package if you
// know what SO_REUSEADDR and SO_REUSEPORT are.
//
// For example:
//
//  // listen on the same port. oh yeah.
//  l1, _ := reuse.Listen("tcp", "127.0.0.1:1234")
//  l2, _ := reuse.Listen("tcp", "127.0.0.1:1234")
//
//  // dial from the same port. oh yeah.
//  l1, _ := reuse.Listen("tcp", "127.0.0.1:1234")
//  l2, _ := reuse.Listen("tcp", "127.0.0.1:1235")
//  c, _ := reuse.Dial("tcp", "127.0.0.1:1234", "127.0.0.1:1235")
//
// Note: cant dial self because tcp/ip stacks use 4-tuples to identify connections,
// and doing so would clash.
package reuseport

import (
	"errors"
	"net"
	"time"
)

// ErrUnsuportedProtocol signals that the protocol is not currently
// supported by this package. This package currently only supports TCP.
var ErrUnsupportedProtocol = errors.New("protocol not yet supported")

// ErrReuseFailed is returned if a reuse attempt was unsuccessful.
var ErrReuseFailed = errors.New("reuse failed")

// Listen listens at the given network and address. see net.Listen
// Returns a net.Listener created from a file discriptor for a socket
// with SO_REUSEPORT and SO_REUSEADDR option set.
func Listen(network, address string) (net.Listener, error) {
	return listenStream(network, address)
}

// ListenPacket listens at the given network and address. see net.ListenPacket
// Returns a net.Listener created from a file discriptor for a socket
// with SO_REUSEPORT and SO_REUSEADDR option set.
func ListenPacket(network, address string) (net.PacketConn, error) {
	return listenPacket(network, address)
}

// Dial dials the given network and address. see net.Dialer.Dial
// Returns a net.Conn created from a file discriptor for a socket
// with SO_REUSEPORT and SO_REUSEADDR option set.
func Dial(network, laddr, raddr string) (net.Conn, error) {

	var d Dialer
	if laddr != "" {
		netladdr, err := ResolveAddr(network, laddr)
		if err != nil {
			return nil, err
		}
		d.D.LocalAddr = netladdr
	}

	// there's a rare case where dial returns successfully but for some reason the
	// RemoteAddr is not yet set. We wait here a while until it is, and if too long
	// passes, we fail.
	c, err := dial(d.D, network, raddr)
	if err != nil {
		return nil, err
	}

	for start := time.Now(); c.RemoteAddr() == nil; {
		if time.Now().Sub(start) > time.Second {
			c.Close()
			return nil, ErrReuseFailed
		}

		<-time.After(20 * time.Microsecond)
	}
	return c, nil
}

// Dialer is used to specify the Dial options, much like net.Dialer.
// We simply wrap a net.Dialer.
type Dialer struct {
	D net.Dialer
}

// Dial dials the given network and address. see net.Dialer.Dial
// Returns a net.Conn created from a file discriptor for a socket
// with SO_REUSEPORT and SO_REUSEADDR option set.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return dial(d.D, network, address)
}
