package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/tangyanhan/sshd/pkg/sshd"
	"github.com/tangyanhan/sshd/pkg/sshd/config"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "./config.toml", "Path to the config file")
	flag.Parse()
	var cfg config.SshConfig
	if err := config.NewSshConfig(configFile, &cfg); err != nil {
		log.Fatalln("Failed to load config file from", configFile, err)
	}
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		cancel()
		log.WithField("signal", sig).Info("Shutting down server")
		os.Exit(0)
	}()

	sshdServer, err := sshd.New(ctx, &cfg)

	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(sshdServer.ListenAndServe())
}
