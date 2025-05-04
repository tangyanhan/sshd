package sshd

import (
	"fmt"
	"testing"
)

func TestSshd(t *testing.T) {
	cfg, err := LoadSSHDConfig("/etc/ssh/sshd_config")
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range cfg {
		fmt.Println("Key=", k, "Value=", v)
	}
}
