package main

import (
	bpmutilsshared "bpm-utils-shared"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	flag "github.com/spf13/pflag"
	yaml "gopkg.in/yaml.v3"
)

var directory = flag.StringP("directory", "D", "", "Path to package directory")
var name = flag.StringP("name", "n", "", "Set the package name")
var description = flag.StringP("description", "d", "Default Package Description", "Set the description")
var version = flag.StringP("version", "v", "1.0", "Set the package version")
var url = flag.StringP("url", "u", "", "Set the package URL")
var license = flag.StringP("license", "l", "", "Set the package licenses")
var template = flag.StringP("template", "t", "gnu-configure", "Set the package template")
var git = flag.BoolP("git", "g", true, "Create git repository")

func main() {
	// Setup flags and help
	setupFlagsAndHelp("bpm-setup <options>", "Sets up files and directories for BPM source package creation")

	// Show command help if no directory name is given
	if *directory == "" {
		log.Println("Error: directory flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Set package name to directory name if empty
	if *name == "" {
		*name = filepath.Base(*directory)
	}

	// run checks
	runChecks()

	// Show summary
	showSummary()

	// Confirmation prompt
	fmt.Printf("Create package directory? [y\\N]: ")
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(text)) != "y" && strings.TrimSpace(strings.ToLower(text)) != "yes" {
		fmt.Println("Cancelling package directory creation...")
		os.Exit(1)
	}

	// Create package directory
	createDirectory()
}

func runChecks() {
	if strings.TrimSpace(*directory) == "" {
		log.Fatalf("No directory was specified!")
	}

	if strings.TrimSpace(*name) == "" {
		log.Fatalf("No package name was specified!")
	}

	if stat, err := os.Stat(path.Join("/etc/bpm-utils/templates/", *template)); err != nil || stat.IsDir() {
		log.Fatalf("%s is not a valid template file!", *template)
	}
}

func showSummary() {
	absPath, err := filepath.Abs(strings.TrimSpace(*directory))
	if err != nil {
		log.Fatalf("Failed to determine absolute path: %s", err)
	}
	fmt.Printf("Setting up package directory at %s with the following information:\n", absPath)
	fmt.Printf("Package name: %s\n", strings.TrimSpace(*name))
	fmt.Printf("Package description: %s\n", *description)
	fmt.Printf("Package version: %s\n", *version)
	if url != nil && *url != "" {
		fmt.Printf("Package URL: %s\n", *url)
	} else {
		fmt.Printf("Package URL: Not Set\n")
	}
	if license != nil && *license != "" {
		fmt.Printf("Package license: %s\n", *license)
	} else {
		fmt.Printf("Package license: Not Set\n")
	}
	fmt.Printf("Template file: %s\n", *template)
	fmt.Printf("Create git repository: %t\n", *git)
}

func replaceVariables(templateContents string) string {
	templateContents = strings.ReplaceAll(templateContents, "$NAME", *name)
	templateContents = strings.ReplaceAll(templateContents, "$DESCRIPTION", *description)
	templateContents = strings.ReplaceAll(templateContents, "$VERSION", *version)
	templateContents = strings.ReplaceAll(templateContents, "$URL", *url)

	return templateContents
}

func createDirectory() {
	// Trim spaces
	*directory = strings.TrimSpace(*directory)
	*name = strings.TrimSpace(*name)

	// Create directory
	err := os.Mkdir(*directory, 0755)
	if err != nil {
		log.Fatalf("Error: could not create directory: %s", err)
	}

	// Create package info struct
	pkgInfo := bpmutilsshared.PackageInfo{
		Name:        *name,
		Description: *description,
		Version:     *version,
		Url:         *url,
		License:     *license,
		Arch:        "any",
		Type:        "source",
		Downloads: []bpmutilsshared.PackageDownload{
			{
				Url:                    "https://wwww.my-url.com/file.tar.gz",
				ExtractTo:              "${BPM_SOURCE}",
				ExtractStripComponents: 1,
				Checksum:               "replaceme",
			},
		},
	}

	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	encoder.Encode(&pkgInfo)

	// Write package info to file
	err = os.WriteFile(path.Join(*directory, "pkg.info"), buffer.Bytes(), 0644)
	if err != nil {
		log.Fatalf("Could not write to pkg.info: %s", err)
	}

	// Create source-files directory
	err = os.Mkdir(path.Join(*directory, "source-files"), 0755)
	if err != nil {
		log.Fatalf("Error: could not create directory: %s", err)
	}

	// Copy source template file
	input, err := os.ReadFile(path.Join("/etc/bpm-utils/templates", *template))
	if err != nil {
		log.Fatalf("Error: could not read template file: %s", err)
		return
	}

	// Replace variables in template file
	input = []byte(replaceVariables(string(input)))

	err = os.WriteFile(path.Join(*directory, "source.sh"), input, 0644)
	if err != nil {
		log.Fatalf("Error: could not write to source file: %s", err)
		return
	}

	// Create git repository
	if git != nil && *git {
		err = exec.Command("git", "init", *directory).Run()
		if err != nil {
			log.Fatalf("Error: could not initialize git repository: %s", err)
		}

		// Copy default gitignore
		defaultGitignoreFile, err := os.Open("/etc/bpm-utils/gitignore.default")
		if err != nil {
			return
		}
		defer defaultGitignoreFile.Close()
		newGitignoreFile, err := os.OpenFile(path.Join(*directory, ".gitignore"), os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Error: could not create .gitignore file: %s", err)
		}

		_, err = io.Copy(newGitignoreFile, defaultGitignoreFile)
		if err != nil {
			log.Fatalf("Error: could not copy data to .gitignore file: %s", err)
		}
	}
}

func setupFlagsAndHelp(usage, desc string) {
	flag.Usage = func() {
		fmt.Println("Usage: " + usage)
		fmt.Println("Description: " + desc)
		fmt.Println("Options:")
		flag.PrintDefaults()
	}
	flag.Parse()
}
