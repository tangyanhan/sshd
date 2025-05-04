package sshd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/gliderlabs/ssh"
)

type ResponseType = byte

const (
	RTOk      ResponseType = 0
	RTWarning ResponseType = 1
	RTError   ResponseType = 2
	RTCreate  ResponseType = 'C'
	RTTime    ResponseType = 'T'
)

// Ack writes an `Ack` message to the remote, does not await its response, a seperate call to ParseResponse is
// therefore required to check if the acknowledgement succeeded.
func Ack(writer io.Writer) error {
	var msg = []byte{0}
	n, err := writer.Write(msg)
	if err != nil {
		return err
	}
	if n < len(msg) {
		return errors.New("failed to write ack buffer")
	}
	return nil
}

func (s *Server) ExecSession(session ssh.Session) {
	user := userFromSession(session)
	if user == nil {
		session.Exit(1)
		return
	}

	log := logFromSession(session)

	commands := session.Command()
	log.Infof("ExecSession commands=%v", commands)
	switch commands[0] {
	case "scp":
		if err := handleScpCommand(session, commands, user); err != nil {
			log.Errorf("scp failed:%v", err)
			fmt.Fprintln(session, RTError, err.Error())
			_ = session.Exit(1)
			return
		}
	default:
		session.Exit(1)
		return
	}
}

var ErrInvalidScpCommand = errors.New("invalid scp command")

func handleScpCommand(session ssh.Session, commands []string, user *SessionUser) error {
	if len(commands) < 3 {
		return ErrInvalidScpCommand
	}

	isUpload := false
	filePath := ""
	for _, flag := range commands {
		switch flag {
		case "scp":
			continue
		case "-t":
			isUpload = true
		case "-f":
			isUpload = false
		default:
			filePath = flag
		}
	}

	log := logFromSession(session)

	if isUpload {
		log.Infof("Upload file to %q", filePath)
		body := make([]byte, 0, 64)
		if _, err := session.Read(body); err != nil {
			return fmt.Errorf("failed to read from client:%v", err)
		}
		log.Info("Received from client:", string(body))
		// Upload file
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file at remote location %s", filePath)
		}
		defer file.Close()
		if err := file.Chown(user.UID, user.GID); err != nil {
			return fmt.Errorf("failed to chown of %q:%v", filePath, err)
		}

		session.Write([]byte{RTOk})

		go func() {
			buf := make([]byte, 1024)
			for {
				n, _ := session.Read(buf)
				if n != 0 {
					log.Info("Message from client:", string(buf[:n]))
				}
			}
		}()
		var buf bytes.Buffer
		buf.WriteByte(RTCreate)
		buf.WriteString("0644 ")
		buf.WriteString("0 ")
		buf.WriteString(filePath)
		buf.WriteByte('\n')
		session.Write(buf.Bytes())
		log.Infof("Start copy file to %s", filePath)
		_, err = io.Copy(file, session)
		return err
	}
	// Download
	log.Infof("Download file from %q", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file from %q:%v", filePath, err)
	}
	defer file.Close()
	_, err = io.Copy(session, file)
	return err
}
