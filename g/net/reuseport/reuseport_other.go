// +build !windows,!linux,!darwin,!dragonfly,!freebsd,!netbsd,!openbsd

package reuseport

// See net.RawConn.Control
//func Control(network, address string, c syscall.RawConn) (err error) {
//	return nil
//}