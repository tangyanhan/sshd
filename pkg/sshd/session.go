package sshd

import (
	"fmt"

	"github.com/gliderlabs/ssh"
)

// Type is the type of SSH session.
type Type string

const (
	// SessionTypeShell is the session's type returned when the SSH client requests a shell.
	SessionTypeShell Type = "shell"
	// SessionTypeHeredoc is the session's type returned when the SSH client requests a command execution with a heredoc.
	// "heredoc" is a format that does not require a TTY, but attaches the client input to the command's stdin.
	// It is used to execute a sequence of commands in a single SSH connection without the need to open a shell.
	SessionTypeHeredoc Type = "heredoc"
	// SessionTypeExec is the session's type returned when the SSH client requests a command execution.
	SessionTypeExec Type = "exec"
	// SessionTypeSubsystem is the session's type returned when the SSH client requests a subsystem.
	SessionTypeSubsystem Type = "subsystem"
	// SessionTypeUnknown is the session's type returned when the SSH client requests an unknown session type.
	SessionTypeUnknown Type = "unknown"
)

// GetSessionType returns the session's type based on the SSH client session.
func GetSessionType(session ssh.Session) (Type, error) {
	_, _, isPty := session.Pty()

	requestType, ok := session.Context().Value("request_type").(string)
	if !ok {
		return SessionTypeUnknown, fmt.Errorf("failed to get request type from session context")
	}

	switch {
	case isPty && requestType == RequestTypeShell:
		return SessionTypeShell, nil
	case !isPty && requestType == RequestTypeShell:
		return SessionTypeHeredoc, nil
	case requestType == RequestTypeExec:
		return SessionTypeExec, nil
	case requestType == RequestTypeSubsystem:
		return SessionTypeSubsystem, nil
	default:
		return SessionTypeUnknown, nil
	}
}
