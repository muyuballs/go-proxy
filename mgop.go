package main

import (
	"context"
	"log"
	"os"

	gp "github.com/muyuballs/go-proxy/core"
)

func main() {
	log.Println("PID:", os.Getpid())
	gp.Main(context.Background(), nil, os.Args...)
}
