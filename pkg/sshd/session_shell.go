package sshd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
)

func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

func (s *Server) ShellSession(session ssh.Session) {
	userVal := session.Context().Value(ctxKeySessionUser)
	if userVal == nil {
		session.Exit(1)
		return
	}
	user := userVal.(*SessionUser)
	ptyReq, winCh, isPty := session.Pty()
	if isPty {
		cmd := exec.Command("/bin/bash")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: uint32(user.UID),
				Gid: uint32(user.GID),
			},
		}
		cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
		f, err := pty.Start(cmd)
		if err != nil {
			panic(err)
		}
		go func() {
			for win := range winCh {
				setWinsize(f, win.Width, win.Height)
			}
		}()
		go func() {
			io.Copy(f, session) // stdin
		}()
		go func() {
			io.Copy(session, f) // stdout
		}()
		s.AddCmd(session.Context().SessionID(), cmd)
		cmd.Wait()
	} else {
		io.WriteString(session, "No PTY requested.\n")
		session.Exit(1)
	}
}
