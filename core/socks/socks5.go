package socks

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/muyuballs/go-proxy/core/common"
	"io"
	"net"
	"strconv"
)

const (
	SocksVer5 = 0x05

	socksCmdConnect            = 0x01
	cmdNotSupport              = 0x07
	NO_AUTHENTICATION_REQUIRED = 0x00
	NO_ACCEPTABLE_METHODS      = 0xFF
	ATYP_IP4                   = 0x01
	ATYP_DOMAIN                = 0x03
	ATYP_IP6                   = 0x04
)

const (
	SocksVer4        = 0x04
	REQUEST_REJECTED = 0x5B
)

/*
			  X'00' NO AUTHENTICATION REQUIRED
	          X'01' GSSAPI
	          X'02' USERNAME/PASSWORD
	          X'03' to X'7F' IANA ASSIGNED
	          X'80' to X'FE' RESERVED FOR PRIVATE METHODS
	          X'FF' NO ACCEPTABLE METHODS
*/

func HandleSocks5(conn io.ReadWriter) (target string, err error) {
	defer func() {
		if f, ok := conn.(common.Flusher); ok {
			f.Flush()
		}
	}()
	ver, err := common.ReadByte(conn)
	if err != nil {
		return
	}
	if ver != SocksVer5 {
		return "", fmt.Errorf("not supported version:%v", ver)
	}
	methodCount, err := common.ReadByte(conn)
	if err != nil {
		return
	}
	methods := make([]byte, methodCount)
	_, err = io.ReadFull(conn, methods)
	if err != nil {
		return
	}
	var hasNAQ bool = false
	for _, n := range methods {
		hasNAQ = n == 0x00
		if hasNAQ {
			break
		}
	}
	if hasNAQ {
		//SELECTED NO AUTHENTICATION REQUIRED
		conn.Write([]byte{SocksVer5, NO_AUTHENTICATION_REQUIRED})
	} else {
		conn.Write([]byte{SocksVer5, NO_ACCEPTABLE_METHODS})
		return "", errors.New("client not support NAQ")
	}
	if f, ok := conn.(common.Flusher); ok {
		f.Flush()
	}
	ver, err = common.ReadByte(conn)
	if err != nil {
		return
	}
	if ver != SocksVer5 {
		return "", errors.New("socks ver must to be 0x05")
	}
	cmd, err := common.ReadByte(conn)
	if err != nil {
		return
	}
	if cmd != socksCmdConnect {
		conn.Write([]byte{SocksVer5, cmdNotSupport})
		return "", errors.New("not supported command")
	}
	_, err = common.ReadByte(conn) //skip RSV byte
	if err != nil {
		return
	}
	atyp, err := common.ReadByte(conn)
	if err != nil {
		return
	}
	var host string
	var port uint16
	if atyp == ATYP_IP4 {
		buf := make([]byte, net.IPv4len+2)
		_, err = io.ReadFull(conn, buf)
		if err != nil {
			return
		}
		host = net.IP(buf[:net.IPv4len]).String()
		port = binary.BigEndian.Uint16(buf[net.IPv4len:])
	} else if atyp == ATYP_IP6 {
		buf := make([]byte, net.IPv6len+2)
		_, err = io.ReadFull(conn, buf)
		if err != nil {
			return
		}
		host = net.IP(buf[:net.IPv6len]).String()
		port = binary.BigEndian.Uint16(buf[net.IPv6len:])
	} else if atyp == ATYP_DOMAIN {
		domainLength, err := common.ReadByte(conn)
		if err != nil {
			return "", err
		}
		buf := make([]byte, domainLength+2)
		_, err = io.ReadFull(conn, buf)
		if err != nil {
			return "", err
		}
		host = string(buf[0:domainLength])
		port = binary.BigEndian.Uint16(buf[domainLength:])
	} else {
		return "", errors.New("not supported address type")
	}
	target = net.JoinHostPort(host, strconv.Itoa(int(port)))
	if err == nil {
		_, err = conn.Write([]byte{SocksVer5, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00})
	}
	return
}
