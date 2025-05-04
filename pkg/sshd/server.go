package sshd

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/gliderlabs/ssh"
	log "github.com/sirupsen/logrus"
	"github.com/tangyanhan/sshd/pkg/sshd/config"
	gossh "golang.org/x/crypto/ssh"
)

type Server struct {
	ctx    context.Context
	server *ssh.Server

	keepAliveInterval time.Duration

	Sessions sync.Map

	cmdLock sync.RWMutex
	cmds    map[string]*exec.Cmd

	sshdConfig SshdConfig
}

func (s *Server) AddCmd(id string, cmd *exec.Cmd) {
	s.cmdLock.Lock()
	defer s.cmdLock.Unlock()
	s.cmds[id] = cmd
}

func (s *Server) RemoveCmd(id string) (*exec.Cmd, bool) {
	s.cmdLock.Lock()
	defer s.cmdLock.Unlock()
	cmd, ok := s.cmds[id]
	if !ok {
		return nil, false
	}
	delete(s.cmds, id)
	return cmd, true
}

func New(ctx context.Context, cfg *config.SshConfig) (*Server, error) {
	sv := &Server{
		ctx:               ctx,
		keepAliveInterval: time.Duration(cfg.KeepAliveSeconds) * time.Second,
		cmds:              make(map[string]*exec.Cmd),
	}

	var hostSigner gossh.Signer

	if cfg.SshdConfigFile != "" {
		sshdConfig, err := LoadSSHDConfig(cfg.SshdConfigFile)
		if err != nil {
			return nil, fmt.Errorf("invalid config file at %q:%v", cfg.SshdConfigFile, err)
		}
		sv.sshdConfig = sshdConfig

		if hostKeyFile, ok := sshdConfig["HostKey"]; ok {
			keyData, err := os.ReadFile(hostKeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read private key file: %w", err)
			}

			// Decode the PEM block
			block, _ := pem.Decode(keyData)
			if block == nil {
				return nil, fmt.Errorf("failed to decode PEM block containing private key:%v", block)
			}

			// Parse the private key
			key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key: %w", err)
			}
			signer, err := gossh.NewSignerFromKey(key)
			if err != nil {
				return nil, fmt.Errorf("failed to create signer:%v", err)
			}
			hostSigner = signer
			log.Info("Loaded host key from:", hostKeyFile)
		}
	} else {
		log.Println("Generate a custom host key")
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("failed to generate rsa key:%v", err)
		}
		signer, err := gossh.NewSignerFromKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to create signer:%v", err)
		}
		hostSigner = signer
	}

	addr := cfg.SshdConfig.Address + ":" + strconv.Itoa(cfg.SshdConfig.Port)
	log.Println("Listening on:", addr)
	sv.server = &ssh.Server{
		Banner:                 cfg.Banner,
		Addr:                   addr,
		SessionRequestCallback: sv.sessionRequestCallback,
		Handler:                sv.sessionHandler,
		HostSigners:            []ssh.Signer{hostSigner},
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": sv.SftpHandler,
		},
		ConnCallback: func(ctx ssh.Context, conn net.Conn) net.Conn {
			closeCallback := func(id string) {
				cmd, exists := sv.RemoveCmd(id)
				if exists {
					_ = cmd.Process.Kill()
				}
			}

			return &sshConn{conn, closeCallback, ctx}
		},
		LocalPortForwardingCallback: func(_ ssh.Context, _ string, _ uint32) bool {
			return true
		},
		ReversePortForwardingCallback: func(_ ssh.Context, _ string, _ uint32) bool {
			return true
		},
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"session":      ssh.DefaultSessionHandler,
			"direct-tcpip": ssh.DirectTCPIPHandler,
		},
	}
	return sv, nil
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

// List of request types that are supported by SSH.
//
// Once the session has been set up, a program is started at the remote end.  The program can be a shell, an application
// program, or a subsystem with a host-independent name.  Only one of these requests can succeed per channel.
//
// Check www.ietf.org/rfc/rfc4254.txt at section 6.5 for more information.
const (
	// RequestTypeShell is the request type for shell.
	RequestTypeShell = "shell"
	// RequestTypeExec is the request type for exec.
	RequestTypeExec = "exec"
	// RequestTypeSubsystem is the request type for any subsystem.
	RequestTypeSubsystem = "subsystem"
	// RequestTypeUnknown is the request type for unknown.
	//
	// It is not a valid request type documentated by SSH's RFC, but it can be useful to identify the request type when
	// it is not known.
	RequestTypeUnknown = "unknown"
)

func (s *Server) sessionRequestCallback(session ssh.Session, requestType string) bool {
	session.Context().SetValue("request_type", requestType)

	go s.startKeepAliveLoop(session)

	ch := make(chan ssh.Signal, 1)
	session.Signals(ch)

	go func() {
		log := logFromSession(session)
		for msg := range ch {
			log.Info("Received message from client:", msg)
		}
	}()

	return true
}

// startKeepAlive sends a keep alive message to the server every in keepAliveInterval seconds.
func (s *Server) startKeepAliveLoop(session ssh.Session) {
	ticker := time.NewTicker(s.keepAliveInterval)
	defer ticker.Stop()

	log.WithFields(log.Fields{
		"interval": s.keepAliveInterval,
	}).Debug("Starting keep alive loop")

loop:
	for {
		select {
		case <-ticker.C:
			if conn, ok := session.Context().Value(ssh.ContextKeyConn).(gossh.Conn); ok {
				if _, _, err := conn.SendRequest("keepalive", false, nil); err != nil {
					log.Error(err)
				}
			}
		case <-session.Context().Done():
			log.Debug("Stopping keep alive loop after session closed")
			ticker.Stop()

			break loop
		}
	}
}

func (s *Server) sessionHandler(session ssh.Session) {
	log.Info("New session request")

	user, err := user.Lookup(session.User())
	if err != nil {
		log.WithError(err).Error("failed to get the user")
		return
	}

	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		log.WithError(err).Error("failed to get the user ID")
		return
	}

	gid, err := strconv.Atoi(user.Gid)
	if err != nil {
		log.WithError(err).Error("failed to get the group IP")

		return
	}

	session.Context().SetValue(ctxKeySessionUser, &SessionUser{
		Username: user.Username,
		UID:      uid,
		GID:      gid,
	})

	if ssh.AgentRequested(session) {

		l, err := ssh.NewAgentListener()
		if err != nil {
			log.WithError(err).Error("failed to create agent listener")

			return
		}

		defer l.Close()

		authSock := l.Addr().String()

		// NOTE: When the agent is started by the root user, we need to change the ownership of the Unix socket created
		// to allow access for the logged-in user.
		if err := os.Chown(path.Dir(authSock), uid, gid); err != nil {
			log.WithError(err).Error("failed to change the permission of directory where unix socket was created")

			return
		}

		if err := os.Chown(authSock, uid, gid); err != nil {
			log.WithError(err).Error("failed to change the permission of unix socket")

			return
		}

		session.Context().SetValue("SSH_AUTH_SOCK", authSock)

		go ssh.ForwardAgentConnections(l, session)
	}

	sessionType, err := GetSessionType(session)
	if err != nil {
		log.Error(err)

		return
	}

	logger := log.WithFields(log.Fields{
		"SessionID": session.Context().SessionID(),
		"User":      user.Username,
		"UID":       uid,
		"GID":       gid,
		"Type":      sessionType,
	})
	sessionWithLog(session, logger)
	logger.Info("Session start")
	switch sessionType {
	case SessionTypeShell:
		s.ShellSession(session)
	case SessionTypeHeredoc:
		log.Error("HereDoc not supported")
	case SessionTypeExec:
		log.Info("Exec command=", session.Command(), "rawCommand=", session.RawCommand(), "perm=", session.Permissions())
		s.ExecSession(session)
	default:
		log.Error("Unknown session type:", sessionType)
	}

	logger.Info("Session ended")
}
