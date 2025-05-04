package sshd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type SshdConfig map[string]string

var regSshdKVPair = regexp.MustCompile(`^(\w+)\s*(.*)\s*$`)

// LoadSSHDConfig 加载并解析 sshd_config 文件
func LoadSSHDConfig(filePath string) (SshdConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件: %w", err)
	}
	defer file.Close()

	config := make(SshdConfig)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		matches := regSshdKVPair.FindStringSubmatch(line)

		if len(matches) != 3 {
			continue
		}

		key := matches[1]
		value := matches[2]

		// 存储到配置中
		config[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件时出错: %w", err)
	}

	return config, nil
}
