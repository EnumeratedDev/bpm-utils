package main

import (
	bpmutilsshared "bpm-utils-shared"
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var directory = flag.String("D", "", "Path to package directory (required)")
var name = flag.String("n", "", "Set the package name")
var description = flag.String("d", "Default Package Description", "Set the description")
var version = flag.String("v", "1.0", "Set the package version")
var url = flag.String("u", "", "Set the package URL")
var license = flag.String("l", "", "Set the package licenses")
var template = flag.String("t", "gnu-configure", "Set the package template")
var git = flag.Bool("g", true, "Create git repository")

func main() {
	// Setup flags and help
	bpmutilsshared.SetupHelp("bpm-setup <options>", "Sets up files and directories for BPM source package creation")
	bpmutilsshared.SetupFlags()

	// Show command help if no directory name is given
	if *directory == "" {
		bpmutilsshared.ShowHelp()
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

	// Create pkg.info contents string
	pkgInfo := "name: " + *name + "\n"
	pkgInfo += "description: " + *description + "\n"
	pkgInfo += "version: " + *version + "\n"
	if url != nil && *url != "" {
		pkgInfo += "url: " + *url + "\n"
	}
	if license != nil && *license != "" {
		pkgInfo += "license: " + *license + "\n"
	}
	pkgInfo += "architecture: any\n"
	pkgInfo += "type: source\n"

	// Write string to file
	err = os.WriteFile(path.Join(*directory, "pkg.info"), []byte(pkgInfo), 0644)
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
	}
}
