package http

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"github.com/google/easypki/pkg/certificate"
	"github.com/google/easypki/pkg/store"
	"github.com/hashicorp/golang-lru"
	"gopkg.in/google/easypki.v1/pkg/easypki"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const Year = time.Hour * 24 * 1
const RootCertificateName = "root"
const StoreDir = "easypki"

var (
	commonSubject = pkix.Name{
		Organization: []string{"SOT DO NOT TRUST"},
	}
	certCache, _ = lru.New(5000)
	genlock      = &sync.Mutex{}
	pki          *easypki.EasyPKI
	caBundle     *certificate.Bundle
)

func InitCertCache(cache string) (err error) {
	p := filepath.Join(cache, StoreDir)
	_, err = os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(p, 0711)
			if err != nil {
				return err
			}
		} else {
			return
		}
	}
	pki = &easypki.EasyPKI{
		Store: &store.Local{Root: p},
	}
	root, e := pki.GetCA(RootCertificateName)
	if e != nil {
		log.Println(err)
		log.Println("root not found ,create it")
		caRequest := &easypki.Request{
			Name: RootCertificateName,
			Template: &x509.Certificate{
				Subject:    commonSubject,
				NotAfter:   time.Now().AddDate(100, 0, 0),
				MaxPathLen: 1,
				IsCA:       true,
			},
		}
		caRequest.Template.Subject.CommonName = "SOT DO NOT TRUST CA"
		if e := pki.Sign(nil, caRequest); e != nil {
			return e
		}
		root, e := pki.GetCA(RootCertificateName)
		if e != nil {
			return e
		}
		caBundle = root
	} else {
		caBundle = root
	}
	return
}

func genCertificate(domain string) (cert *tls.Certificate, err error) {
	parts := strings.Split(domain, ".")
	baseDomain := "*." + strings.Join(parts[len(parts)-2:], ".")
	log.Println(baseDomain)
	if t, ok := certCache.Get(baseDomain); ok {
		log.Println(baseDomain, "certificate found from lru cache")
		return t.(*tls.Certificate), nil
	}
	srv, err := createCertificate(baseDomain)
	if err != nil {
		return nil, err
	}
	cert = &tls.Certificate{
		Certificate: [][]byte{srv.Cert.Raw, caBundle.Cert.Raw},
		PrivateKey:  srv.Key,
		Leaf:        srv.Cert,
	}
	certCache.Add(baseDomain, cert)
	return
}

func createCertificate(domain string) (srv *certificate.Bundle, err error) {
	genlock.Lock()
	defer genlock.Unlock()
	srv, err = pki.GetBundle(RootCertificateName, domain)
	if err == nil {
		log.Println(domain, "certificate found from local store")
		return
	}
	srvRequest := &easypki.Request{
		Name: domain,
		Template: &x509.Certificate{
			Subject:  commonSubject,
			NotAfter: time.Now().AddDate(10, 0, 0),
			DNSNames: []string{domain},
		},
		PrivateKeySize: 4096,
	}
	srvRequest.Template.Subject.CommonName = domain
	if err := pki.Sign(caBundle, srvRequest); err != nil {
		log.Printf("Sign(%v, %v): go error: %v != expected nil\n", caBundle, srvRequest, err)
		return nil, err
	}
	srv, err = pki.GetBundle(RootCertificateName, srvRequest.Name)
	if err != nil {
		log.Printf("GetBundle(%v, %v): go error %v != expected nil", "root", srvRequest.Name, err)
		return
	}
	return
}
