package http

import (
	"github.com/valyala/fasthttp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func handleRedirect(fullUrl string, ctx *fasthttp.RequestCtx) bool {
	ri := GetMappedPath(fullUrl)
	if ri == nil {
		return false
	}
	ctx.SetConnectionClose()
	if ri.Type == RedirectCustom {
		code, _ := strconv.Atoi(ri.Target)
		if ri.Headers != nil {
			for k, v := range ri.Headers {
				ctx.Response.Header.Set(k, v)
			}
		}
		if ri.ContentType != "" {
			ctx.SetContentType(ri.ContentType)
		} else {
			ctx.SetContentType("text/plain")
		}
		ctx.SetStatusCode(code)
		ctx.SetBodyString(ri.Body)
		return true
	}

	if ri.Type == RedirectFile {
		fi, err := os.Open(ri.Target)
		if err != nil {
			if ri.Fallback == FallbackToSource {
				return false
			}
			ctx.Error("Redirect To File:"+err.Error(), 404)
			return true
		}
		ctx.SetBodyStream(fi, -1)
		return true
	}

	if ri.Type == RedirectFolder {
		i := strings.Index(fullUrl, "?")
		if i < 0 {
			i = strings.Index(fullUrl, "#")
		}
		if i < 0 {
			i = 0
		}
		subPath := strings.TrimPrefix(fullUrl[i:], ri.Url)
		fullPath := filepath.Join(ri.Target, subPath)
		fi, err := os.Open(fullPath)
		if err != nil {
			if ri.Fallback == FallbackToSource {
				return false
			}
			ctx.Error("Redirect To Folder:"+err.Error(), 404)
			return true
		}
		ctx.SetBodyStream(fi, -1)
		return true
	}
	return false
}
