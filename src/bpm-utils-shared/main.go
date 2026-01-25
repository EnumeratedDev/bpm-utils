package bpm_utils_shared

import (
	"os"

	"gopkg.in/yaml.v3"
)

type BPMUtilsConfig struct {
	PrivilegeEscalatorCmd string `yaml:"privilege_escalator_cmd"`
	DefaultMaintainer     string `yaml:"default_maintainer,omitempty"`
}

func ReadBPMUtilsConfig() (*BPMUtilsConfig, error) {
	data, err := os.ReadFile("/etc/bpm-utils/bpm-utils.conf")
	if err != nil {
		return nil, err
	}

	config := &BPMUtilsConfig{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
