package http

import (
	"bufio"
	"github.com/muyuballs/go-proxy/core/common"
	"github.com/valyala/fasthttp"
	"io"
	"log"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	HopByHops = []string{"Proxy-Connection", "Connection", "Proxy-Authenticate", "Keep-Alive"}
)

func trimRequestHeader(ctx *fasthttp.RequestCtx) {
	reqHeader := ctx.Request.Header
	for _, h := range HopByHops {
		reqHeader.Del(h)
	}
	reqHeader.SetConnectionClose()
	ctx.Request.Header = reqHeader
}

func copyHttpPayload(ctx *fasthttp.RequestCtx, sessionInfo *SessionInfo, rconn *common.ACStream, conf *common.Config) {
	var sessionReqCache = ""
	var sessionRespCache = ""

	if conf.SessionCacheDir != "" {
		sessionReqCache = filepath.Join(conf.SessionCacheDir, sessionInfo.Sid) + ".ss"
		if !conf.OnlyCacheRequest {
			sessionRespCache = filepath.Join(conf.SessionCacheDir, sessionInfo.Sid) + ".sr"
		}
	}
	if sessionReqCache != "" || sessionRespCache != "" {
		jrw, err := common.NewJRW(rconn.Origin().(io.ReadWriteCloser), sessionRespCache, sessionReqCache)
		if err != nil {
			log.Println(err)
			ctx.Error(err.Error(), fasthttp.StatusServiceUnavailable)
			sessionInfo.SessionDone()
			return
		}
		rconn.Destroy()
		rconn = common.NewACS(jrw)
	}
	defer func() {
		_ = rconn.Close()
	}()
	_, _ = ctx.Request.WriteTo(rconn)
	rconn.Flush()
	err := ctx.Response.Header.Read(rconn.Reader())
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusServiceUnavailable)
		sessionInfo.SessionDone()
		return
	}
	sessionInfo.ResponseInfo = &ResponseInfo{
		Status:      ctx.Response.StatusCode(),
		Version:     "HTTP/1.1",
		Message:     fasthttp.StatusMessage(ctx.Response.StatusCode()),
		BodySize:    ctx.Response.Header.ContentLength(),
		ContextType: string(ctx.Response.Header.ContentType()),
		Headers:     make(map[string]string),
	}
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		sessionInfo.ResponseInfo.Headers[string(key)] = string(value)
	})
	cl := ctx.Response.Header.ContentLength()
	for _, h := range HopByHops {
		ctx.Response.Header.Del(h)
	}
	startTime := time.Now()
	var copySize int64
	var copyErr error
	defer func() {
		cost := time.Since(startTime)
		log.Printf("%v %v %v %v/s %v --> %v\n", "IN", copySize, common.FormatNS(float64(copySize)), common.FormatNS(float64(copySize)/cost.Seconds()), cost, copyErr)
		sessionInfo.SessionDone()
	}()
	switch cl {
	case -1: //chunk
		xrconn := rconn.Open()
		ctx.SetConnectionClose()
		ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
			defer func() {
				_ = xrconn.Close()
				_ = w.Flush()
			}()
			for {
				data, _, err := xrconn.ReadLine()
				if err != nil {
					log.Println(err)
					return
				}
				if strings.TrimSpace(string(data)) == "0" {
					break
				}
				cs, err := strconv.ParseInt(string(data), 16, 32)
				if err != nil {
					log.Println(err)
					return
				}
				cs, copyErr := io.CopyN(w, xrconn, cs)
				if copyErr != nil {
					return
				}
				copySize += cs
				_, err = xrconn.Discard(2)
				if err != nil {
					log.Println(err)
					return
				}
			}
		})
	case -2: //EOF
		ctx.SetBodyStream(rconn.Open(), -1)
	case 0:
		return
	default: //Fix len
		ctx.SetBodyStream(rconn.Open(), cl)
	}
}

func buildSessionInfo(conf *common.Config, ctx *fasthttp.RequestCtx) *SessionInfo {
	sessionInfo := NewSessionInfo(conf)
	if taddr, ok := ctx.RemoteAddr().(*net.TCPAddr); ok {
		sessionInfo.RemoteAddr = taddr.IP.String()
		sessionInfo.RemotePort = taddr.Port
	}
	sessionInfo.RequestInfo = &RequestInfo{
		Host:     string(ctx.Host()),
		Method:   string(ctx.Method()),
		Version:  "HTTP/1.1",
		Protocol: "HTTP",
		FullUrl:  BuildFullUrl("http", string(ctx.Host()), string(ctx.RequestURI())),
		Url:      TrimHttpPrefix(string(ctx.RequestURI())),
		Headers:  make(map[string]string),
		Query:    make(map[string]string),
		WebForm:  make(map[string]string),
		Files:    make([]*FileInfo, 0),
	}
	reqUrl := string(ctx.RequestURI())
	if strings.HasPrefix(reqUrl, "http://") {
		sessionInfo.RequestInfo.Url = string(ctx.RequestURI()[strings.Index(string(ctx.RequestURI()[HttpPrefixLen:]), "/")+HttpPrefixLen:])
	} else {
		sessionInfo.RequestInfo.Url = reqUrl
		sessionInfo.RequestInfo.Protocol = "HTTPS"
		sessionInfo.RequestInfo.FullUrl = "https://" + sessionInfo.RequestInfo.Host + sessionInfo.RequestInfo.Url
	}
	return sessionInfo
}

func fitSessionInfo(sessionInfo *SessionInfo, ctx *fasthttp.RequestCtx) {
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		sessionInfo.RequestInfo.Headers[string(key)] = string(value)
	})
	ctx.QueryArgs().VisitAll(func(key, value []byte) {
		sessionInfo.RequestInfo.Query[string(key)] = string(value)
	})
	form, err := ctx.MultipartForm()
	if err == nil {
		for key, v := range form.Value {
			sessionInfo.RequestInfo.WebForm[key] = strings.Join(v, ",")
		}
		for key, v := range form.File {
			if len(v) > 0 {
				fi := &FileInfo{
					Key:         key,
					Name:        v[0].Filename,
					Size:        int(v[0].Size),
					ContentType: v[0].Header.Get("Content-Type"),
				}
				sessionInfo.RequestInfo.Files = append(sessionInfo.RequestInfo.Files, fi)
			}
		}
	} else {
		ctx.PostArgs().VisitAll(func(key, value []byte) {
			sessionInfo.RequestInfo.WebForm[string(key)] = string(value)
		})
	}
	sessionInfo.RequestInfo.ContentType = string(ctx.Request.Header.ContentType())
}
