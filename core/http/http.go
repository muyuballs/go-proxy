package http

import (
	"crypto/tls"
	"fmt"
	"github.com/muyuballs/go-proxy/core/common"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

const (
	HttpPrefixLen = len("http://")
	HttpPort      = 80
	HttpsPort     = 443
)

var (
	helloTemplate, _ = template.New("hello").Parse(helloPage)
	SslVersionMap    = map[byte]string{
		0: "SSL 3.0",
		1: "TlS 1.0",
		2: "TlS 1.1",
		3: "TlS 1.2",
	}
	HopByHops = []string{"Proxy-Connection", "Connection", "Proxy-Authenticate", "Keep-Alive"}
	_server   = &fasthttp.Server{
		Name: "sot",
	}
)

func TrimHttpPrefix(url string) string {
	if strings.HasPrefix(url, "https://") {
		url = url[HttpPrefixLen:]
	} else if strings.HasPrefix(url, "http://") {
		url = url[HttpsPrefixLen:]
	}
	first := strings.Index(url, "/")
	if first < 0 {
		first = 0
	}
	return url[first:]
}

func BuildFullUrl(protocol, host, path string) string {
	if path[0] != '/' {
		return path
	}
	return fmt.Sprintf("%s://%s%s", protocol, host, path)
}

func InitHandler(conf *common.Config) {
	_server.Handler = httpHandler(conf)
	initHttpsHandler(conf)
}

func HandleHttp(acs *common.ACStream) (err error) {
	return _server.ServeConn(acs)
}

func hostToTcpAddr(host string, defPort int) (addr string, err error) {
	target, err := url.Parse("SOT://" + host)
	if err != nil {
		return
	}
	port, err := strconv.Atoi(target.Port())
	if err != nil {
		port = defPort
	}
	return fmt.Sprintf("%s:%d", target.Hostname(), port), nil
}

func httpHandler(conf *common.Config) func(*fasthttp.RequestCtx) {
	certPath := conf.HelloPageUrl + "/do-not-trust.crt"
	if strings.HasSuffix(conf.HelloPageUrl, "/") {
		certPath = conf.HelloPageUrl + "do-not-trust.crt"
	}
	return func(ctx *fasthttp.RequestCtx) {
		defer func() {
			log.Println("ctx done")
		}()
		if "CONNECT" == string(ctx.Method()) {
			target, err := hostToTcpAddr(string(ctx.Host()), HttpsPort)
			if err != nil {
				log.Println(err)
				ctx.Error(err.Error(), fasthttp.StatusServiceUnavailable)
				return
			}
			log.Println(target)
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.Hijack(func(lconn net.Conn) {
				lacs := common.NewACS(lconn)
				defer func() {
					_ = lacs.Close()
				}()
				v, err := lacs.Pick(6)
				if err != nil {
					log.Println(err)
					return
				}
				if conf.DecryptHttps {
					if v[0] == 0x16 && v[1] == 0x03 && v[2] <= 3 && v[5] == 0x01 {
						log.Println("ssl", SslVersionMap[v[2]], " handshake")
						err := handleHttps(common.NewACS(tls.Server(lacs.Open(), &tls.Config{
							GetCertificate: func(info *tls.ClientHelloInfo) (certificate *tls.Certificate, e error) {
								return genCertificate(info.ServerName)
							},
						})))
						if err != nil {
							log.Println(err)
						}
						return
					}
				}
				sessionInfo := buildSessionInfo(conf, ctx)
				sessionInfo.RequestInfo.FullUrl = BuildFullUrl("https", string(ctx.Host()), string(ctx.RequestURI()))
				sessionInfo.RequestInfo.Protocol = "TUNNEL"
				rconn, err := common.DialRemote(conf, nil, target)
				if err != nil {
					log.Println(err)
					return
				}
				racs := common.NewACS(rconn)
				defer func() {
					_ = racs.Close()
				}()
				go common.Transfer(racs.Open(), lacs.Open(), "OUT")
				common.Transfer(lacs.Open(), racs.Open(), "IN")
				sessionInfo.SessionDone()
			})
		} else {
			sessionInfo := buildSessionInfo(conf, ctx)
			fitSessionInfo(sessionInfo, ctx)
			if sessionInfo.RequestInfo.FullUrl == certPath {
				log.Println("request proxy root cert")
				ctx.SetContentType("application/x-x509-ca-cert")
				_, _ = ctx.Write(caBundle.Cert.Raw)
				return
			}
			if sessionInfo.RequestInfo.FullUrl == conf.HelloPageUrl {
				log.Println("request proxy hello page")
				ctx.SetContentType("text/html;charset=utf8")
				headers := make(map[string]interface{})
				ctx.Request.Header.VisitAll(func(key, value []byte) {
					headers[string(key)] = string(value)
				})
				_ = helloTemplate.Execute(ctx, map[string]interface{}{
					"ServerName": conf.ServerName,
					"Headers":    headers,
					"TimeStamp":  time.Now(),
					"CertPath":   certPath,
				})
				sessionInfo.SessionDone()
				return
			}
			target, err := hostToTcpAddr(string(ctx.Host()), HttpPort)
			if err != nil {
				ctx.Error(err.Error(), http.StatusServiceUnavailable)
				sessionInfo.SessionDone()
				return
			}
			reqPath := GetMappedPath(sessionInfo.RequestInfo.FullUrl)
			if strings.HasPrefix(reqPath, "file://") {
				log.Println("send local file")
				fi, err := os.Open(reqPath)
				if err != nil {
					ctx.Error(err.Error(), http.StatusInternalServerError)
					sessionInfo.SessionDone()
					return
				}
				ctx.SetBodyStream(fi, -1)
				return
			}
			reqHeader := ctx.Request.Header
			reqHeader.SetRequestURIBytes([]byte(TrimHttpPrefix(reqPath)))
			for _, h := range HopByHops {
				reqHeader.Del(h)
			}
			reqHeader.SetConnectionClose()
			ctx.Request.Header = reqHeader
			rconn, err := common.DialRemote(conf, nil, target)
			if err != nil {
				log.Println(err)
				ctx.Error(err.Error(), fasthttp.StatusServiceUnavailable)
				sessionInfo.SessionDone()
				return
			}
			copyHttpPayload(ctx, sessionInfo, rconn, conf)
		}
	}
}
