package main

import (
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

var directory = flag.String("D", "", "Path to package directory")
var name = flag.String("n", "", "Set the package name (Defaults to \"package-name\")")
var description = flag.String("d", "Default Package Description", "Set the description (Defaults to \"package-description\")")
var version = flag.String("v", "1.0", "Set the package version (Defaults to \"1.0\")")
var url = flag.String("u", "", "Set the package URL (Optional)")
var license = flag.String("l", "", "Set the package licenses (Optional)")
var template = flag.String("t", "source.default", "Set the package template (Defaults to \"source.default\")")
var git = flag.Bool("g", true, "Create git repository (Defaults to true)")

func main() {
	// Setup flags
	setupFlags()

	// Show command help if no directory name is given
	if *directory == "" {
		help()
		os.Exit(1)
	}

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

func setupFlags() {
	flag.Usage = help
	flag.Parse()
	if *name == "" {
		*name = *directory
	}
}

func help() {
	fmt.Println("Usage: bpm-setup <options>")
	fmt.Println("Options:")
	fmt.Println("  -D=<directory> | Path to package directory")
	fmt.Println("  -n=<name> | Set the package name (Defaults to \"package-name\")")
	fmt.Println("  -d=<description> | Set the package description (Defaults to \"Default package description\")")
	fmt.Println("  -v=<version> | Set the package version (Defaults to \"1.0\")")
	fmt.Println("  -u=<url> | Set the package URL (Optional)")
	fmt.Println("  -l=<licenses> | Set the package licenses (Optional)")
	fmt.Println("  -t=<template file> | Use a template file (Defaults to source.default)")
	fmt.Println("  -g=<true/false> | Create git repository (Defaults to true)")
}

func showSummary() {
	absPath, err := filepath.Abs(*directory)
	if err != nil {
		log.Fatalf("Failed to determine absolute path: %s", err)
	}
	fmt.Printf("Setting up package directory at %s with the following information:\n", absPath)
	fmt.Printf("Package name: %s\n", *name)
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

func createDirectory() {
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
	input, err := os.ReadFile(path.Join("/etc/bpm-utils/", *template))
	if err != nil {
		log.Fatalf("Error: could not read template file: %s", err)
		return
	}
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
