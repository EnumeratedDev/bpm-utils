package bpm_utils_shared

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/drone/envsubst"
	"gopkg.in/yaml.v3"
)

type PackageInfo struct {
	Name            string            `yaml:"name"`
	Description     string            `yaml:"description"`
	Version         string            `yaml:"version"`
	Revision        int               `yaml:"revision,omitempty"`
	Url             string            `yaml:"url,omitempty"`
	License         string            `yaml:"license,omitempty"`
	Arch            string            `yaml:"architecture,omitempty"`
	OutputArch      string            `yaml:"output_architecture,omitempty"`
	Type            string            `yaml:"type,omitempty"`
	Keep            []string          `yaml:"keep,omitempty"`
	Depends         []string          `yaml:"depends,omitempty"`
	OptionalDepends []string          `yaml:"optional_depends,omitempty"`
	MakeDepends     []string          `yaml:"make_depends,omitempty"`
	Conflicts       []string          `yaml:"conflicts,omitempty"`
	Replaces        []string          `yaml:"replaces,omitempty"`
	Provides        []string          `yaml:"provides,omitempty"`
	Downloads       []PackageDownload `yaml:"downloads,omitempty"`
	SplitPackages   []*PackageInfo    `yaml:"split_packages,omitempty"`
}

type PackageDownload struct {
	Url                    string `yaml:"url"`
	Type                   string `yaml:"type,omitempty"`
	NoExtract              bool   `yaml:"no_extract,omitempty"`
	ExtractToBPMSource     bool   `yaml:"extract_to_bpm_source,omitempty"`
	ExtractStripComponents int    `yaml:"extract_strip_components,omitempty"`
	GitBranch              string `yaml:"git_branch,omitempty"`
	Filepath               string `yaml:"filepath,omitempty,omitempty"`
	Checksum               string `yaml:"checksum,omitempty"`
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
		OutputArch:      "",
		Type:            "",
		Keep:            make([]string, 0),
		Depends:         make([]string, 0),
		MakeDepends:     make([]string, 0),
		OptionalDepends: make([]string, 0),
		Conflicts:       make([]string, 0),
		Replaces:        make([]string, 0),
		Provides:        make([]string, 0),
		Downloads:       make([]PackageDownload, 0),
		SplitPackages:   make([]*PackageInfo, 0),
	}

	// Unmarshal yaml
	err = yaml.Unmarshal(output, pkgInfo)
	if err != nil {
		return nil, err
	}

	return pkgInfo, nil
}

func ReadPacakgeInfoFromFile(path string) (*PackageInfo, error) {
	// Extract package info using tar
	output, err := os.ReadFile(path)
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
		OutputArch:      "",
		Type:            "",
		Keep:            make([]string, 0),
		Depends:         make([]string, 0),
		OptionalDepends: make([]string, 0),
		MakeDepends:     make([]string, 0),
		Conflicts:       make([]string, 0),
		Replaces:        make([]string, 0),
		Provides:        make([]string, 0),
		Downloads:       make([]PackageDownload, 0),
		SplitPackages:   make([]*PackageInfo, 0),
	}

	// Unmarshal yaml
	err = yaml.Unmarshal(output, pkgInfo)
	if err != nil {
		return nil, err
	}

	return pkgInfo, nil
}

func (pkgDownload *PackageDownload) CalculateChecksum(pkgInfo *PackageInfo) (string, error) {
	switch pkgDownload.Type {
	case "", "file":
		fmt.Println("Downloading and calculating checksum for file...")

		// Replace variables in download url
		downloadUrl := pkgDownload.Url
		downloadUrl, err := envsubst.Eval(downloadUrl, func(s string) string {
			switch s {
			case "BPM_PKG_VERSION":
				return pkgInfo.Version
			case "BPM_PKG_NAME":
				return pkgInfo.Name
			default:
				return ""
			}
		})
		if err != nil {
			return "", err
		}

		cmd := exec.Command("sh", "-c", fmt.Sprintf("curl -s -L %s | sha256sum | awk '{print $1}'", downloadUrl))
		cmd.Stderr = os.Stderr

		checksum, err := cmd.Output()
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(checksum)), err
	case "git":
		fmt.Println("Calculating checksum for git branch...")

		// Replace variables in git branch
		gitBranch := pkgDownload.GitBranch
		gitBranch, err := envsubst.Eval(gitBranch, func(s string) string {
			switch s {
			case "BPM_PKG_VERSION":
				return pkgInfo.Version
			case "BPM_PKG_NAME":
				return pkgInfo.Name
			default:
				return ""
			}
		})
		if err != nil {
			return "", err
		}

		if pkgDownload.GitBranch == "" {
			return "", fmt.Errorf("'git_branch' field cannot be empty")
		}

		cmd := exec.Command("sh", "-c", fmt.Sprintf("git ls-remote --refs %s | grep 'refs/.*/%s$' | awk '{print $1}'", pkgDownload.Url, gitBranch))
		cmd.Stderr = os.Stderr

		checksum, err := cmd.Output()
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(checksum)), err
	default:
		return "", fmt.Errorf("unknown download type (%s)", pkgDownload.Type)
	}
}
