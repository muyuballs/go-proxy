package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"github.com/muyuballs/go-proxy/core/common"
	"log"
	"math/big"
	"net"
)

func StartServer(conf *common.Config) (err error) {
	var tlsConfig *tls.Config
	if conf.Certificate != "" && conf.CertKey != "" {
		xcert, err := tls.LoadX509KeyPair(conf.Certificate, conf.CertKey)
		if err != nil {
			return err
		}
		tlsConfig = &tls.Config{Certificates: []tls.Certificate{xcert}}
	} else {
		tlsConfig, err = generateTLSConfig()
		if err != nil {
			return err
		}
	}
	l, err := net.Listen("tcp", conf.Listen)
	if err != nil {
		return err
	}
	for {
		session, err := l.Accept()
		if err != nil {
			log.Println(err)
			return err
		}
		if tcpConn, ok := session.(*net.TCPConn); ok {
			tcpConn.SetNoDelay(true)
		}
		go handSession(tls.Server(session, tlsConfig))
	}
	return nil
}

func handSession(ses net.Conn) {
	log.Println("new session", ses.RemoteAddr())
	acs := common.NewACS(ses)
	defer acs.Close()
	var tl uint32
	err := binary.Read(acs, binary.BigEndian, &tl)
	if err != nil {
		log.Println(err)
		return
	}
	buf := make([]byte, tl)
	err = common.ReadFull(acs, buf)
	if err != nil {
		log.Println(err)
		return
	}
	target := string(buf)
	log.Println("Target:", target)
	conn, err := net.Dial("tcp", target)
	if err != nil {
		log.Println(err)
		return
	}
	cAcs := common.NewACS(conn)
	defer cAcs.Close()
	go common.Transfer(cAcs.Open(), acs.Open(), "IN")
	go common.Transfer(acs.Open(), cAcs.Open(), "OUT")
}

func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}, nil
}
