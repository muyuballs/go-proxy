package core

import (
	"fmt"
	"github.com/muyuballs/go-proxy/core/client"
	"github.com/muyuballs/go-proxy/core/common"
	"github.com/muyuballs/go-proxy/core/http"
	"github.com/muyuballs/go-proxy/core/server"
	"log"
	"os"
	"reflect"
	"time"

	"context"
	"github.com/urfave/cli"
)

func startService(conf *common.Config) error {
	go func() {
		for {
			select {
			case <-conf.Context.Done():
				return
			case <-time.Tick(15 * time.Second):
				common.PrintAcs()
			}
		}
	}()
	log.Println("======================")
	val := reflect.ValueOf(*conf)
	tpe := reflect.TypeOf(*conf)
	for i := 0; i < val.NumField(); i++ {
		if val.Field(i).CanInterface() {
			log.Printf("%-20s : %v\n", tpe.Field(i).Name, val.Field(i).Interface())
		}
	}
	log.Println("======================")
	if !conf.ServerMode {
		return client.StartClient(conf)
	} else {
		return server.StartServer(conf)
	}
}

func Main(ctx context.Context, logChan chan interface{}, args ...string) {
	myApp := cli.NewApp()
	myApp.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "listen",
			Value: "0.0.0.0:8999",
			Usage: "proxy port",
		},
		cli.StringFlag{
			Name:  "certificate",
			Value: "",
			Usage: "server ssl certificate,(default auto gen)",
		},
		cli.StringFlag{
			Name:  "cert-key",
			Value: "",
			Usage: "server ssl certificate key,(default auto gen)",
		},
		cli.StringFlag{
			Name:  "log",
			Value: "-",
			Usage: "log file,- for stdout",
		},
		cli.BoolFlag{
			Name:  "server",
			Usage: "server mode",
		},
		cli.IntFlag{
			Name:  "read-timeout",
			Value: 30,
		},
		cli.IntFlag{
			Name:  "write-timeout",
			Value: 30,
		},
		cli.IntFlag{
			Name:  "idle-timeout",
			Value: 30,
		},
		cli.StringFlag{
			Name:  "local-only-list,lol",
			Value: "",
		},
		cli.StringFlag{
			Name:  "remote",
			Usage: "run in client mode with remote server",
			Value: "", //ss.yuelwish.top:60000
		},
		cli.BoolFlag{
			Name:  "disable-http",
			Usage: "disable http proxy",
		},
		cli.StringFlag{
			Name:  "session-cache-dir",
			Usage: "http session cache dir, default is disable",
			Value: "",
		},
		cli.BoolFlag{
			Name:  "only-cache-request",
			Usage: "only cache request info",
		},
		cli.BoolFlag{
			Name:  "decrypt-https",
			Usage: "decrypt https",
		},
		cli.StringFlag{
			Name:  "cert-cache-dir",
			Usage: " cert cache dir path",
			Value: os.TempDir(),
		},
		cli.StringFlag{
			Name:  "hello-page-url",
			Usage: "proxy hello page url",
			Value: "http://sot.sot/",
		},
		cli.StringFlag{
			Name:  "server-name",
			Usage: "server name",
			Value: "Sot",
		},
	}
	myApp.Action = func(c *cli.Context) (err error) {
		conf := &common.Config{
			LogFlags:         log.LstdFlags | log.LUTC,
			ReadTimeout:      c.Int("read-timeout"),
			WriteTimeout:     c.Int("write-timeout"),
			IdleTimeout:      c.Int("idle-timeout"),
			Listen:           c.String("listen"),
			Certificate:      c.String("certificate"),
			CertKey:          c.String("cert-key"),
			LolFile:          c.String("local-only-list"),
			ServerMode:       c.Bool("server"),
			Remote:           c.String("remote"),
			LogFile:          c.String("log"),
			HttpEnable:       !c.Bool("enable-http"),
			CacheDir:         c.String("cache-dir"),
			LogChan:          logChan,
			Context:          ctx,
			SessionCacheDir:  c.String("session-cache-dir"),
			OnlyCacheRequest: c.Bool("only-cache-request"),
			DecryptHttps:     c.Bool("decrypt-https"),
			CertCache:        c.String("cert-cache-dir"),
			HelloPageUrl:     c.String("hello-page-url"),
			ServerName:       c.String("server-name"),
		}
		if conf.DecryptHttps {
			if err := http.InitCertCache(conf.CertCache); err != nil {
				return err
			}
		}
		if conf.Context == nil {
			conf.Context = context.Background()
		}
		if "-" != conf.LogFile {
			logOut, err := os.OpenFile(c.String("log"), os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_SYNC, 0755)
			if err != nil {
				return err
			}
			defer logOut.Close()
			defer logOut.Sync()
			conf.LogOut = logOut
		} else {
			conf.LogOut = os.Stdout
		}
		return startService(conf)
	}
	fmt.Println(myApp.Run(args))
}
