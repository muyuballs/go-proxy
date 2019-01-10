package common

import (
	"context"
	"io"
)

type Config struct {
	HelloPageUrl     string
	ServerName       string
	LogFlags         int
	LogOut           io.Writer
	LogFile          string
	innerProxy       string
	Listen           string
	Certificate      string
	CertKey          string
	ReadTimeout      int
	IdleTimeout      int
	WriteTimeout     int
	LolFile          string
	ServerMode       bool
	Remote           string
	HttpEnable       bool
	CacheDir         string
	LogChan          chan interface{}
	Context          context.Context
	SessionCacheDir  string
	OnlyCacheRequest bool
	DecryptHttps     bool
	CertCache        string
}
