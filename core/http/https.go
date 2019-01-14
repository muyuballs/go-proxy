package http

import (
	"crypto/tls"
	"github.com/muyuballs/go-proxy/core/common"
	"github.com/valyala/fasthttp"
	"log"
	"net"
	"net/http"
)

var (
	HttpsPrefixLen = len("http://")
	_httpsServer   = &fasthttp.Server{
		Name: "sot-https",
	}
)

func initHttpsHandler(conf *common.Config) {
	_httpsServer.Handler = httpsHandler(conf)
}
func HandleHttps(acs *common.ACStream) (err error) {
	return handleHttps(common.NewACS(tls.Server(acs, &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (certificate *tls.Certificate, e error) {
			return genCertificate(info.ServerName)
		},
	})))
}

func handleHttps(acs *common.ACStream) (err error) {
	return _httpsServer.ServeConn(acs)
}

func httpsHandler(conf *common.Config) func(*fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		defer func() {
			log.Println("ctx done")
		}()
		sessionInfo := buildSessionInfo(conf, ctx)
		if "CONNECT" == string(ctx.Method()) {
			sessionInfo.RequestInfo.FullUrl = BuildFullUrl("https", string(ctx.Host()), string(ctx.RequestURI()))
			sessionInfo.RequestInfo.Protocol = "TUNNEL"
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
			fitSessionInfo(sessionInfo, ctx)
			target, err := hostToTcpAddr(string(ctx.Host()), HttpsPort)
			if err != nil {
				ctx.Error(err.Error(), http.StatusServiceUnavailable)
				sessionInfo.SessionDone()
				return
			}
			if handleRedirect(sessionInfo.RequestInfo.FullUrl, ctx) {
				return
			}
			trimRequestHeader(ctx)
			rconn, err := common.DialRemote(conf, nil, target)
			if err != nil {
				log.Println(err)
				ctx.Error(err.Error(), fasthttp.StatusServiceUnavailable)
				sessionInfo.SessionDone()
				return
			}
			xrconn := common.NewACS(tls.Client(rconn.Origin().(net.Conn), &tls.Config{
				InsecureSkipVerify: true,
			}))
			rconn.Destroy()
			copyHttpPayload(ctx, sessionInfo, xrconn, conf)
		}
	}
}
