package main

import (
	"flag"
	"fmt"
	"os"
)

var directory = flag.String("D", "", "Path to package directory")
var name = flag.String("n", "", "Set the package name (Defaults to \"package-name\")")
var description = flag.String("d", "Default Package Description", "Set the description (Defaults to \"package-description\")")
var version = flag.String("v", "1.0", "Set the package version (Defaults to \"1.0\")")
var url = flag.String("u", "", "Set the package URL (Optional)")
var license = flag.String("l", "", "Set the package licenses (Optional)")
var template = flag.String("t", "source.default", "Set the package template (Defaults to \"source.default\")")
var git = flag.String("g", "", "Create git repository (Defaults to true)")

func main() {
	// Setup flags
	setupFlags()

	// Show command help if no directory name is given
	if *directory == "" {
		help()
		os.Exit(1)
	}
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
