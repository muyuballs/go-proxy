package client

import (
	"github.com/muyuballs/go-proxy/core/common"
	"github.com/muyuballs/go-proxy/core/http"
	"github.com/muyuballs/go-proxy/core/socks"
	"log"
	"net"
)

func handClient(conf *common.Config, conn net.Conn) {
	acs := common.NewACS(conn)
	defer acs.Close()
	v, err := acs.Pick(1)
	if err != nil {
		log.Println(err)
		return
	}
	//http.HandleHttps(acs.Open())
	var remote string
	if v[0] == socks.SocksVer4 {
		remote, err = socks.HandleSocks4(acs)
		if err != nil {
			log.Println(err)
			return
		}
	} else if v[0] == socks.SocksVer5 {
		remote, err = socks.HandleSocks5(acs)
		if err != nil {
			log.Println(err)
			return
		}
	} else if conf.HttpEnable && v[0] >= 'A' && v[0] <= 'Z' {
		err := http.HandleHttp(acs.Open())
		if err != nil {
			log.Println(err)
		}
		return
	} else {
		log.Println("unsupported socks version")
		return
	}
	log.Println("target:", remote)
	rAcs, err := common.DialRemote(conf, nil, remote)
	if err != nil {
		_ = conn.Close()
		log.Println(err)
		return
	}
	defer rAcs.Close()
	go common.Transfer(rAcs.Open(), acs.Open(), "OUT")
	go common.Transfer(acs.Open(), rAcs.Open(), "IN")
}

func StartClient(conf *common.Config) error {
	err := common.LoadLol(conf)
	if err != nil {
		return err
	}

	l, err := net.Listen("tcp", conf.Listen)
	if err != nil {
		return err
	}
	http.InitHandler(conf)
	defer l.Close()
	for {
		select {
		case <-conf.Context.Done():
			return nil
		default:
			conn, err := l.Accept()
			if err != nil {
				log.Println(err)
				continue
			}
			go handClient(conf, conn)

		}
	}
}
