package main

import (
	"context"
	"flag"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/controller"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"
)

func main() {
	time.Sleep(5 * time.Second) //wait for routing tables in cluster

	confLocation := flag.String("config", "config.json", "configuration file")
	flag.Parse()

	config, err := configuration.Load(*confLocation)
	if err != nil {
		log.Fatal("ERROR: unable to load config ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	_, err = controller.New(config, ctx)
	if err != nil {
		debug.PrintStack()
		log.Fatal("FATAL:", err)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	sig := <-shutdown
	log.Println("received shutdown signal", sig)
	cancel()
	time.Sleep(1 * time.Second) //give connections time to close gracefully
}
