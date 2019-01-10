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

func skipIDEN(conn io.Reader) error {
	buf := make([]byte, 0)
	for {
		b, err := common.ReadByte(conn)
		if err != nil {
			return err
		}
		buf = append(buf, b)
		if b == 0x00 {
			return nil
		}
	}
}

func HandleSocks4(conn io.ReadWriter) (target string, err error) {
	defer func() {
		if f, ok := conn.(common.Flusher); ok {
			f.Flush()
		}
	}()
	ver, err := common.ReadByte(conn)
	if err != nil {
		return
	}
	if ver != SocksVer4 {
		return "", fmt.Errorf("not supported version:%v", ver)
	}
	cmd, err := common.ReadByte(conn)
	if err != nil {
		return
	}
	if cmd != socksCmdConnect {
		conn.Write([]byte{SocksVer4, REQUEST_REJECTED, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return "", errors.New("not supported command")
	}
	buf := make([]byte, 2)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return
	}
	port := binary.BigEndian.Uint16(buf)
	buf = make([]byte, net.IPv4len)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return
	}
	if buf[0] == 0x00 && buf[1] == 0x00 && buf[2] == 0x00 { //socks4a ip address
		err = skipIDEN(conn)
		if err != nil {
			return
		}
		buf = make([]byte, 0)
		for {
			b, err := common.ReadByte(conn)
			if err != nil {
				return "", err
			}
			if b == 0x00 {
				break
			}
			buf = append(buf, b)
		}
		target = net.JoinHostPort(string(buf), strconv.Itoa(int(port)))
	} else {
		target = net.JoinHostPort(net.IP(buf).String(), strconv.Itoa(int(port)))
		err = skipIDEN(conn)
	}
	if err == nil {
		_, err = conn.Write([]byte{0x00, 0x5A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	}
	return
}
