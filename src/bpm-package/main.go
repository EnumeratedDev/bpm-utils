package main

import (
	bpmutilsshared "bpm-utils-shared"
	"bytes"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

var compile = flag.BoolP("compile", "c", false, "Compile BPM source package")
var verbose = flag.BoolP("verbose", "v", false, "Show additional information about the compilation process")
var skipCheck = flag.Bool("skip-checks", false, "Skip 'check' function while compiling")
var keepCompilationFiles = flag.BoolP("keep", "k", false, "Keep compilation files after successful package compilation")
var installDepends = flag.BoolP("depends", "d", false, "Install package dependencies for compilation")
var installPackage = flag.BoolP("install", "i", false, "Install compiled BPM package after compilation finishes")
var compilationJobs = flag.IntP("jobs", "j", 0, "Set the amount of concurrent processes to use for source package compilation")
var updateInfo = flag.BoolP("update-info", "u", false, "Update the info.yml file")
var signPackage = flag.BoolP("sign", "s", false, "Sign package using GPG")
var yesAll = flag.BoolP("yes", "y", false, "Accept all confirmation prompts")

func main() {
	// Setup flags and help
	setupFlagsAndHelp("bpm-package <options>", "Generates source BPM package from current directory")

	// Run checks
	runChecks()

	// Create BPM archive
	outputFile := createArchive()

	if *compile {
		compilePackage(outputFile)
	}

	if repo := bpmutilsshared.GetRepository(); repo != "" {
		bpmutilsshared.UpdateDatabases(repo)
	}
}

func runChecks() {
	// Check if info.yml file exists
	if stat, err := os.Stat("info.yml"); err != nil || !stat.Mode().IsRegular() {
		log.Fatalf("Error: info.yml does not exist or is not a regular file")
	}

	// Check if recipe.sh file exists
	if stat, err := os.Stat("recipe.sh"); err != nil || !stat.Mode().IsRegular() {
		log.Fatalf("Error: info.yml does not exist or is not a regular file")
	}
}

func createArchive() string {
	// Read BPM utils config
	config, err := bpmutilsshared.ReadBPMUtilsConfig()
	if err != nil {
		log.Fatalf("Error: failed to read config: %s", err)
	}

	filesToInclude := make([]string, 0)

	// Include base files
	filesToInclude = append(filesToInclude, "info.yml", "recipe.sh")

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
			fmt.Printf("Package script '%s' found\n", script)
			filesToInclude = append(filesToInclude, script)
		}
	}

	// Read info.yml file
	pkgInfo, err := bpmutilsshared.ReadPacakgeInfoFromFile("info.yml")
	if err != nil {
		log.Fatalf("Error: could not read package info: %s", err)
	}

	// Update info.yml file
	if *updateInfo {
		// Update download checksums
		for i, download := range pkgInfo.Downloads {
			if download.Checksum == "skip" {
				continue
			}

			download.Checksum, err = download.CalculateChecksum(pkgInfo)
			if err != nil {
				log.Fatalf("Could not calculate checksum for download entry %d: %s", i+1, err)
			}

			pkgInfo.Downloads[i] = download
		}

		// Add default maintainer
		if config.AddDefaultMaintainer {
			if !slices.Contains(pkgInfo.Maintainers, config.DefaultMaintainer) {
				pkgInfo.Maintainers = append(pkgInfo.Maintainers, config.DefaultMaintainer)
			}
		}

		// Save yaml back to file
		var data bytes.Buffer
		encoder := yaml.NewEncoder(&data)
		encoder.SetIndent(2)
		encoder.Encode(pkgInfo)
		if err != nil {
			log.Fatalf("Could not marshal package info: %s", err)
		}

		// Stat info.yml
		stat, err := os.Stat("info.yml")
		if err != nil {
			log.Fatalf("Could not stat info.yml: %s", err)
		}

		// Write package info back to file
		err = os.WriteFile("info.yml", data.Bytes(), stat.Mode().Perm())
		if err != nil {
			log.Fatalf("Could not write package info to info.yml: %s", err)
		}
	}

	// Remove old package from source dir
	if repo := bpmutilsshared.GetRepository(); repo != "" {
		if database, err := bpmutilsshared.ReadDatabase(path.Join(repo, "source/database.bpmdb")); err == nil {
			if entry, ok := database.Entries[pkgInfo.Name]; ok {
				pkgFilepath := path.Join(repo, "source", entry.Filepath)
				err := os.Remove(pkgFilepath)
				if err != nil {
					log.Printf("Warning: could not remove old source package (%s): %s", pkgFilepath, err)
				}

				// Remove package signature
				if _, err := os.Stat(pkgFilepath + ".sig"); err == nil {
					err := os.Remove(pkgFilepath + ".sig")
					if err != nil {
						log.Printf("Warning: could not remove old source package signature (%s): %s", pkgFilepath+".str", err)
					}
				}
			}
		}
	}

	// Create filename
	filename := fmt.Sprintf("%s-%s-%d-%s-src.bpm", pkgInfo.Name, pkgInfo.Version, pkgInfo.Revision, pkgInfo.Arch)
	if repo := bpmutilsshared.GetRepository(); repo != "" {
		// Create source directory
		os.MkdirAll(path.Join(repo, "source", pkgInfo.Arch), 0755)

		filename = path.Join(repo, "source", pkgInfo.Arch, filename)
	}

	// Create archive using tar
	cmd := exec.Command("tar", "cf", filename,
		"--sort=name",
		"--pax-option=exthdr.name=%d/PaxHeaders/%f,delete=atime,delete=ctime",
		"--mtime=UTC 1970-01-01",
		"--owner=0", "--group=0", "--numeric-owner")
	cmd.Args = append(cmd.Args, filesToInclude...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Error: failed to create BPM source archive: %s", err)
	}

	// Sign package
	if *signPackage {
		cmd := exec.Command("gpg", "--detach-sign", filename)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		err := cmd.Run()
		if err != nil {
			log.Fatalf("Error: could not sign package: %s", err)
		}
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
	if *verbose {
		args = append(args, "-v")
	}
	if *skipCheck {
		args = append(args, "-s")
	}
	if *keepCompilationFiles {
		args = append(args, "-k")
	}
	if *installDepends {
		args = append(args, "-d")
	}
	if *compilationJobs > 0 {
		args = append(args, "-j"+strconv.Itoa(*compilationJobs))
	}
	if *yesAll {
		args = append(args, "-y")
	}
	args = append(args, "--output-fd=3")
	args = append(args, archive)
	cmd := exec.Command("bpm", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set output pipe for file descriptor 3
	cmdOutputReader, cmdOutputWriter, err := os.Pipe()
	if err != nil {
		log.Fatalf("Error: failed to create pipe: %s", err)
	}
	defer cmdOutputReader.Close()
	defer cmdOutputWriter.Close()
	cmd.ExtraFiles = append(cmd.ExtraFiles, cmdOutputWriter)

	// Run command
	err = cmd.Start()
	if err != nil {
		log.Fatalf("Error: failed to compile BPM source package: %s", err)
	}

	// Wait for process to complete
	err = cmd.Wait()
	if err != nil {
		log.Fatalf("Error: failed to compile BPM source package: %s", err)
	}

	// Read cmd output
	cmdOutputWriter.Close()
	cmdOutput, err := io.ReadAll(cmdOutputReader)
	if err != nil {
		log.Fatalf("Error: failed to get cmd output: %s", err)
	}

	// Put output file into slice
	outputPkgs := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(cmdOutput)), "\n") {
		// Read generated package info
		pkgInfo, err := bpmutilsshared.ReadPacakgeInfoFromTarball(line)

		if repo := bpmutilsshared.GetRepository(); repo != "" {
			// Remove old package from binary dir
			if database, err := bpmutilsshared.ReadDatabase(path.Join(repo, "binary/database.bpmdb")); err == nil {
				if entry, ok := database.Entries[pkgInfo.Name]; ok {
					pkgFilepath := path.Join(repo, "binary", entry.Filepath)
					err := os.Remove(pkgFilepath)
					if err != nil {
						log.Printf("Warning: could not remove old binary package (%s): %s", pkgFilepath, err)
					}

					// Remove package signature
					if _, err := os.Stat(pkgFilepath + ".sig"); err == nil {
						err := os.Remove(pkgFilepath + ".sig")
						if err != nil {
							log.Printf("Warning: could not remove old binary package signature (%s): %s", pkgFilepath+".str", err)
						}
					}
				}
			}

			// Move package to binary dir
			if err != nil {
				log.Fatalf("Error: could not read package info: %s", err)
			}
			newPath := path.Join(repo, "binary", pkgInfo.Arch, path.Base(line))
			os.MkdirAll(path.Dir(newPath), 0755)
			os.Rename(line, newPath)
			outputPkgs[pkgInfo.Name] = newPath

		} else {
			outputPkgs[pkgInfo.Name] = line
		}
	}

	// Sign package
	if *signPackage {
		for k, v := range outputPkgs {
			cmd := exec.Command("gpg", "--detach-sign", v)
			cmd.Dir = path.Dir(v)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin

			err := cmd.Run()
			if err != nil {
				log.Fatalf("Error: could not sign package (%s) at: %s", k, v)
			}
		}
	}

	// Print out generated packages
	for k, v := range outputPkgs {
		fmt.Printf("Package (%s) was successfully compiled! Binary package generated at: %s\n", k, v)
	}

	// Install compiled packages
	if *installPackage && len(outputPkgs) != 0 {
		// Read BPM utils config
		config, err := bpmutilsshared.ReadBPMUtilsConfig()
		if err != nil {
			log.Fatalf("Error: failed to read config: %s", err)
		}

		// Setup install command
		args = make([]string, 0)
		args = append(args, "bpm", "install", "--reinstall")
		if *yesAll {
			args = append(args, "-y")
		}

		args = append(args, slices.Collect(maps.Values(outputPkgs))...)
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

func setupFlagsAndHelp(usage, desc string) {
	flag.Usage = func() {
		fmt.Println("Usage: " + usage)
		fmt.Println("Description: " + desc)
		fmt.Println("Options:")
		flag.PrintDefaults()
	}
	flag.Parse()
}
