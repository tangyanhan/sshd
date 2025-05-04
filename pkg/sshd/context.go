package sshd

import (
	"github.com/gliderlabs/ssh"
	"github.com/sirupsen/logrus"
)

const (
	ctxKeySessionUser = "user"
	ctxKeySessionLog  = "log"
)

type SessionUser struct {
	Username string
	UID      int
	GID      int
}

func userFromSession(session ssh.Session) *SessionUser {
	userVal := session.Context().Value(ctxKeySessionUser)
	if userVal == nil {
		return nil
	}
	user := userVal.(*SessionUser)
	return user
}

func logFromSession(session ssh.Session) *logrus.Entry {
	logVal := session.Context().Value(ctxKeySessionLog)
	if logVal == nil {
		return logrus.StandardLogger().WithField("sessionId", session.Context().SessionID())
	}
	return logVal.(*logrus.Entry)
}

func sessionWithLog(session ssh.Session, log *logrus.Entry) {
	session.Context().SetValue(ctxKeySessionLog, log)
}
