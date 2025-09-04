package bpm_utils_shared

import (
	"os/exec"

	"gopkg.in/yaml.v3"
)

type PackageInfo struct {
	Name            string         `yaml:"name,omitempty"`
	Description     string         `yaml:"description,omitempty"`
	Version         string         `yaml:"version,omitempty"`
	Revision        int            `yaml:"revision,omitempty"`
	Url             string         `yaml:"url,omitempty"`
	License         string         `yaml:"license,omitempty"`
	Arch            string         `yaml:"architecture,omitempty"`
	Type            string         `yaml:"type,omitempty"`
	Keep            []string       `yaml:"keep,omitempty"`
	Depends         []string       `yaml:"depends,omitempty"`
	MakeDepends     []string       `yaml:"make_depends,omitempty"`
	OptionalDepends []string       `yaml:"optional_depends,omitempty"`
	Conflicts       []string       `yaml:"conflicts,omitempty"`
	Replaces        []string       `yaml:"replaces,omitempty"`
	Provides        []string       `yaml:"provides,omitempty"`
	SplitPackages   []*PackageInfo `yaml:"split_packages,omitempty"`
}

func ReadPacakgeInfo(path string) (*PackageInfo, error) {
	// Extract package info using tar
	cmd := exec.Command("tar", "-x", "-f", path, "pkg.info", "-O")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	pkgInfo := &PackageInfo{
		Name:            "",
		Description:     "",
		Version:         "",
		Revision:        1,
		Url:             "",
		License:         "",
		Arch:            "",
		Type:            "",
		Keep:            make([]string, 0),
		Depends:         make([]string, 0),
		MakeDepends:     make([]string, 0),
		OptionalDepends: make([]string, 0),
		Conflicts:       make([]string, 0),
		Replaces:        make([]string, 0),
		Provides:        make([]string, 0),
		SplitPackages:   make([]*PackageInfo, 0),
	}

	// Unmarshal yaml
	err = yaml.Unmarshal(output, pkgInfo)
	if err != nil {
		return nil, err
	}

	return pkgInfo, nil
}
