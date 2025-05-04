package sshd

import (
	"fmt"
	"io"

	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
)

func (s *Server) SftpHandler(sess ssh.Session) {
	debugStream := io.Discard
	serverOptions := []sftp.ServerOption{
		sftp.WithDebug(debugStream),
	}

	log.Info("SftpHandler start")
	defer log.Info("SftpHandler done")

	server, err := sftp.NewServer(
		sess,
		serverOptions...,
	)
	if err != nil {
		log.Printf("sftp server init error: %s\n", err)
		return
	}
	if err := server.Serve(); err == io.EOF {
		server.Close()
		fmt.Println("sftp client exited session.")
	} else if err != nil {
		fmt.Println("sftp server completed with error:", err)
	}
}
