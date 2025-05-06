package sshd

import (
	"fmt"
	"os"

	"github.com/gliderlabs/ssh"
	"github.com/sirupsen/logrus"
)

func (s *Server) LoadAuthorizedKeys() error {
	if s.config.SshdConfig.AuthorizedKeysFile == "" {
		return nil
	}

	// Read the public key file
	raw, err := os.ReadFile(s.config.SshdConfig.AuthorizedKeysFile)
	if err != nil {
		return fmt.Errorf("failed to read public key file from %s: %w", s.config.SshdConfig.AuthorizedKeysFile, err)
	}
	log := logrus.WithFields(logrus.Fields{"F": s.config.SshdConfig.AuthorizedKeysFile, "M": "LoadPublicKeys"})
	// Parse the public key file
	publicKeys := make([]ssh.PublicKey, 0)
	for {
		pubKey, comment, options, rest, err := ssh.ParseAuthorizedKey(raw)
		if err != nil {
			break
		}
		log.Info("Loaded public key", "comment", comment, "options", options, "remain=", len(rest))
		if len(rest) > 0 {
			raw = rest
		} else {
			break
		}
		publicKeys = append(publicKeys, pubKey)
	}

	log.Info("Loaded public keys", "count", len(publicKeys))
	s.authorizedKeys = publicKeys
	return nil
}

func (s *Server) PubKeyHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	for _, authorizedKey := range s.authorizedKeys {
		if ssh.KeysEqual(key, authorizedKey) {
			return true
		}
	}
	return false
}

// // PasswordHandler is a callback for performing password authentication.
// type PasswordHandler func(ctx Context, password string) bool
