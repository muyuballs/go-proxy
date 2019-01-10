package common

import (
	"crypto/tls"
	"encoding/binary"
	"log"
	"net"
)

func DialServer(conf *Config) (ses net.Conn, err error) {
	log.Println("Dial remote", conf.Remote)
	ses, err = tls.Dial("tcp", conf.Remote, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		log.Println("dial", err)
		return
	}
	return
}

func DialRemote(conf *Config, laddr *net.TCPAddr, target string) (conn *ACStream, err error) {
	host, port, err := net.SplitHostPort(target)
	if err == nil {
		host = GetMappedHost(host)
		target = net.JoinHostPort(host, port)
	}
	raddr, err := net.ResolveTCPAddr("tcp", target)
	if err != nil {
		return
	}
	if conf.Remote == "" || IsLocalOnly(host) {
		conn, err := net.DialTCP("tcp", laddr, raddr)
		if err != nil {
			return nil, err
		}
		conn.SetNoDelay(true)
		return NewACS(conn), nil
	} else {
		session, err := DialServer(conf)
		if err != nil {
			return nil, err
		}
		acs := NewACS(session)
		defer acs.Close()
		err = binary.Write(acs, binary.BigEndian, uint32(len(target)))
		if err != nil {
			log.Println("write target", err)
			return nil, err
		}
		_, err = acs.Write([]byte(target))
		if err != nil {
			log.Println("write target", err)
			return nil, err
		}
		return acs.Open(), nil
	}
}
