package config

import (
	"github.com/jinzhu/configor"
)

type SshConfig struct {
	Banner           string
	KeepAliveSeconds int `default:"30"`

	// Either take value from sshdConfig or a sshd config file
	SshdConfig     SshdConfig `toml:"sshd"`
	SshdConfigFile string
}

type SshdConfig struct {
	HostKeyFile            string
	Port                   int `default:"22" env:"PORT"`
	Address                string
	PermitRootLogin        bool `default:"false"`
	PasswordAuthentication bool `default:"false"`
	AllowTcpForwarding     bool `default:"false"`
	AuthorizedKeysFile     string
}

func NewSshConfig(file string, cfg *SshConfig) error {
	return configor.Load(cfg, file)
}
