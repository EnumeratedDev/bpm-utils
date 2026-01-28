package bpm_utils_shared

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/drone/envsubst"
	"gopkg.in/yaml.v3"
)

type PackageInfo struct {
	Name            string            `yaml:"name"`
	Description     string            `yaml:"description,omitempty"`
	Version         string            `yaml:"version,omitempty"`
	Revision        int               `yaml:"revision,omitempty"`
	Url             string            `yaml:"url,omitempty"`
	License         string            `yaml:"license,omitempty"`
	Maintainers     []string          `yaml:"maintainers,omitempty"`
	Arch            string            `yaml:"architecture,omitempty"`
	OutputArch      string            `yaml:"output_architecture,omitempty"`
	Type            string            `yaml:"type,omitempty"`
	Keep            []string          `yaml:"keep,omitempty"`
	Depends         []string          `yaml:"depends,omitempty"`
	RuntimeDepends  []string          `yaml:"runtime_depends,omitempty"`
	OptionalDepends []string          `yaml:"optional_depends,omitempty"`
	MakeDepends     []string          `yaml:"make_depends,omitempty"`
	CheckDepends    []string          `yaml:"check_depends,omitempty"`
	Conflicts       []string          `yaml:"conflicts,omitempty"`
	Replaces        []string          `yaml:"replaces,omitempty"`
	Provides        []string          `yaml:"provides,omitempty"`
	Options         []string          `yaml:"options,omitempty"`
	Downloads       []PackageDownload `yaml:"downloads,omitempty"`
	SplitPackages   []*PackageInfo    `yaml:"split_packages,omitempty"`
}

type PackageDownload struct {
	Url      string `yaml:"url"`
	Type     string `yaml:"type,omitempty"`
	Filepath string `yaml:"filepath,omitempty,omitempty"`

	// Archive options
	NoExtract              bool   `yaml:"no_extract,omitempty"`
	ExtractTo              string `yaml:"extract_to,omitempty"`
	ExtractStripComponents int    `yaml:"extract_strip_components,omitempty"`

	// Git options
	CloneTo   string `yaml:"clone_to,omitempty"`
	GitBranch string `yaml:"git_branch,omitempty"`

	Checksum string `yaml:"checksum,omitempty"`
}

func ReadPackageInfo(data []byte) (*PackageInfo, error) {
	pkgInfo := &PackageInfo{
		Revision:        1,
		Keep:            make([]string, 0),
		Depends:         make([]string, 0),
		MakeDepends:     make([]string, 0),
		RuntimeDepends:  make([]string, 0),
		OptionalDepends: make([]string, 0),
		Conflicts:       make([]string, 0),
		Replaces:        make([]string, 0),
		Provides:        make([]string, 0),
		Options:         make([]string, 0),
		Downloads:       make([]PackageDownload, 0),
		SplitPackages:   make([]*PackageInfo, 0),
	}

	// Unmarshal yaml
	err := yaml.Unmarshal(data, pkgInfo)
	if err != nil {
		return nil, err
	}

	return pkgInfo, nil
}

func (pkgInfo *PackageInfo) GetFullVersion() string {
	return pkgInfo.Version + "-" + strconv.Itoa(pkgInfo.Revision)
}

func ReadPacakgeInfoFromTarball(path string) (*PackageInfo, error) {
	// Extract package info using tar
	cmd := exec.Command("tar", "-x", "-f", path, "pkg.info", "-O")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	pkgInfo, err := ReadPackageInfo(output)
	if err != nil {
		return nil, err
	}

	return pkgInfo, nil
}

func ReadPacakgeInfoFromFile(path string) (*PackageInfo, error) {
	// Read data from file
	output, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pkgInfo, err := ReadPackageInfo(output)
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

		cmd := exec.Command("sh", "-c", fmt.Sprintf("git ls-remote -bt %s | grep -E 'refs/.*/%s(\\^\\{\\})?$' | tail -n1 | awk '{print $1}'", pkgDownload.Url, gitBranch))
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
