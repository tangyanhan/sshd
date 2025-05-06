package config

import (
	"fmt"
	"strconv"

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
	if cfg.SshdConfigFile != "" {
		sshdConfigMap, err := LoadSSHDConfig(cfg.SshdConfigFile)
		if err != nil {
			return err
		}
		cfg.SshdConfig.HostKeyFile = sshdConfigMap["HostKey"]
		port, err := strconv.Atoi(sshdConfigMap["Port"])
		if err != nil {
			return fmt.Errorf("invalid port value: %v", err)
		}
		cfg.SshdConfig.Port = port
		cfg.SshdConfig.Address = sshdConfigMap["Address"]
		cfg.SshdConfig.PermitRootLogin, err = strconv.ParseBool(sshdConfigMap["PermitRootLogin"])
		if err != nil {
			return fmt.Errorf("invalid PermitRootLogin value: %v", err)
		}
		cfg.SshdConfig.PasswordAuthentication, err = strconv.ParseBool(sshdConfigMap["PasswordAuthentication"])
		if err != nil {
			return fmt.Errorf("invalid PasswordAuthentication value: %v", err)
		}
		cfg.SshdConfig.AllowTcpForwarding, err = strconv.ParseBool(sshdConfigMap["AllowTcpForwarding"])
		if err != nil {
			return fmt.Errorf("invalid AllowTcpForwarding value: %v", err)
		}
		cfg.SshdConfig.AuthorizedKeysFile = sshdConfigMap["AuthorizedKeysFile"]
	}
	return configor.Load(cfg, file)
}
