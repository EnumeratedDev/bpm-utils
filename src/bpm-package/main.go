package main

import (
	bpm_utils_shared "bpm-utils-shared"
	"bufio"
	"flag"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var compile = flag.Bool("c", false, "Compile BPM source package")
var skipCheck = flag.Bool("s", false, "Skip 'check' function while compiling")
var installDepends = flag.Bool("d", false, "Install package dependencies for compilation")
var installPackage = flag.Bool("i", false, "Install compiled BPM package after compilation finishes")
var yesAll = flag.Bool("y", false, "Accept all confirmation prompts")

func main() {
	// Setup flags
	setupFlags()

	// Run checks
	runChecks()

	// Create BPM archive
	outputFile := createArchive()

	if *compile {
		compilePackage(outputFile)
	}
}

func setupFlags() {
	flag.Usage = help
	flag.Parse()
}

func help() {
	fmt.Println("Usage: bpm-package <options>")
	fmt.Println("Description: Generates source BPM package from current directory")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func runChecks() {
	// Check if pkg.info file exists
	if stat, err := os.Stat("pkg.info"); err != nil || !stat.Mode().IsRegular() {
		log.Fatalf("Error: pkg.info does not exist or is not a regular file")
	}

	// Check if source.sh file exists
	if stat, err := os.Stat("source.sh"); err != nil || !stat.Mode().IsRegular() {
		log.Fatalf("Error: pkg.info does not exist or is not a regular file")
	}
}

func createArchive() string {
	filesToInclude := make([]string, 0)

	// Include base files
	filesToInclude = append(filesToInclude, "pkg.info", "source.sh")

	// Check if non-empty source-files directory exists and include it
	if stat, err := os.Stat("source-files"); err == nil && stat.IsDir() {
		dir, err := os.ReadDir("source-files")
		if err == nil && len(dir) != 0 {
			fmt.Println("Non-empty 'source-files' directory found")
			filesToInclude = append(filesToInclude, "source-files")
		}
	}

	// Check for package scripts and include them
	for _, script := range []string{"pre_install.sh", "post_install.sh", "pre_update.sh", "post_update.sh", "pre_remove.sh", "post_remove.sh"} {
		if stat, err := os.Stat(script); err == nil && stat.Mode().IsRegular() {
			fmt.Printf("Package script '%s' found", script)
			filesToInclude = append(filesToInclude, script)
		}
	}

	// Read pkg.info file into basic struct
	pkgInfo := struct {
		Name     string `yaml:"name"`
		Version  string `yaml:"version"`
		Revision int    `yaml:"revision"`
		Arch     string `yaml:"architecture"`
	}{
		Revision: 1,
	}
	data, err := os.ReadFile("pkg.info")
	if err != nil {
		log.Fatalf("Error: could not read pkg.info file")
	}
	err = yaml.Unmarshal(data, &pkgInfo)
	if err != nil {
		log.Fatalf("Error: could not unmarshal pkg.info file")
	}

	// Create filename
	filename := fmt.Sprintf("%s-%s-%d-%s-src.bpm", pkgInfo.Name, pkgInfo.Version, pkgInfo.Revision, pkgInfo.Arch)

	// Create archive using tar
	args := make([]string, 0)
	args = append(args, "-c", "--owner=0", "--group=0", "--no-same-owner", "-f", filename)
	args = append(args, filesToInclude...)
	cmd := exec.Command("tar", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Error: failed to create BPM source archive: %s", err)
	}

	// Get absolute path to filename
	absFilepath, err := filepath.Abs(filename)
	if err != nil {
		log.Fatalf("Error: failed to get absolute path of BPM source archive: %s", err)
	}
	fmt.Printf("BPM source archive created at: %s\n", absFilepath)

	return absFilepath
}

func compilePackage(archive string) {
	// Setup compile command
	args := make([]string, 0)
	args = append(args, "compile")
	if *skipCheck {
		args = append(args, "-s")
	}
	if *installDepends {
		args = append(args, "-d")
	}
	if *yesAll {
		args = append(args, "-y")
	}
	args = append(args, archive)
	cmd := exec.Command("bpm", args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error: could not setup stdout pipe: %s", err)
	}

	// Run command
	err = cmd.Start()
	if err != nil {
		log.Fatalf("Error: failed to compile BPM source package: %s", err)
	}

	// Print command output and store it in variable
	pipeOutput := ""
	buf := bufio.NewReader(pipe)
	for {
		b, err := buf.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("Error: failed to read byte from command: %s", err)
		}

		pipeOutput += string(b)
		fmt.Print(string(b))
	}
	fmt.Println()

	// Put output file into slice
	outputFiles := make([]string, 0)
	for _, line := range strings.Split(pipeOutput, "\n") {
		if strings.Contains(line, "Binary package generated at: ") {
			path := strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			outputFiles = append(outputFiles, path)
		}
	}

	// Wait for process to complete
	err = cmd.Wait()
	if err != nil {
		log.Fatalf("Error: failed to compile BPM source package: %s", err)
	}

	// Install compiled packages
	if *installPackage && len(outputFiles) != 0 {
		// Read BPM utils config
		config, err := bpm_utils_shared.ReadBPMUtilsConfig()
		if err != nil {
			log.Fatalf("Error: failed to read config: %s", err)
		}

		// Setup install command
		args = make([]string, 0)
		args = append(args, "bpm", "install", "--reinstall")
		if *yesAll {
			args = append(args, "-y")
		}
		args = append(args, outputFiles...)
		cmd = exec.Command(config.PrivilegeEscalatorCmd, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		// Run command
		err = cmd.Run()
		if err != nil {
			log.Fatalf("Error: failed to install compiled BPM packages: %s", err)
		}
	}
}
