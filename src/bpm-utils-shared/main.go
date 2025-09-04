package bpm_utils_shared

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

var usageMsg string
var description string

type BPMUtilsConfig struct {
	PrivilegeEscalatorCmd string `yaml:"privilege_escalator_cmd"`
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

func SetupFlags() {
	flag.Usage = ShowHelp
	flag.Parse()
}

func SetupHelp(usage, desc string) {
	usageMsg = usage
	description = desc
}

func ShowHelp() {
	fmt.Println("Usage: " + usageMsg)
	fmt.Println("Description: " + description)
	fmt.Println("Options:")
	flag.PrintDefaults()
}
