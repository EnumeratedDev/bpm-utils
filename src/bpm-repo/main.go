package main

import (
	bpmutilsshared "bpm-utils-shared"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"log"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

var currentFlagSet *flag.FlagSet

func main() {
	if len(os.Args) < 2 {
		log.Println("Error: no subcommand")
		listSubcommands()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "create-repo", "c":
		// Setup flags and help
		flagset := flag.NewFlagSet("create-repo", flag.ExitOnError)
		setupFlagsAndHelp(flagset, fmt.Sprintf("bpm-repo %s <options>", subcommand), "Create a new BPM repository", os.Args[:1])

		// Get current database
		repo := bpmutilsshared.GetRepository()
		if repo != "" {
			log.Fatal("Error: this command cannot be run inside a BPM repository")
		}

		scanner := bufio.NewReader(os.Stdin)
		fmt.Print("Repository name: ")
		name, err := scanner.ReadString('\n')
		if err != nil {
			log.Fatalf("Error: could not read user input: %s", err)
		}
		fmt.Print("Repository description: ")
		desc, err := scanner.ReadString('\n')
		if err != nil {
			log.Fatalf("Error: could not read user input: %s", err)
		}

		createRepository(strings.TrimSpace(name), strings.TrimSpace(desc))
	case "update-db", "u":
		// Setup flags and help
		flagset := flag.NewFlagSet("update-db", flag.ExitOnError)
		setupFlagsAndHelp(flagset, fmt.Sprintf("bpm-repo %s <options>", subcommand), "Update update source and binary databases in current repository", os.Args[:1])

		// Get current database
		repo := bpmutilsshared.GetRepository()
		if repo == "" {
			log.Fatal("Error: this command may only be run inside a BPM repository")
		}

		bpmutilsshared.UpdateDatabases(repo)
	case "list", "l":
		flagset := flag.NewFlagSet("list", flag.ExitOnError)
		setupFlagsAndHelp(flagset, fmt.Sprintf("bpm-repo %s <options>", subcommand), "List packages", os.Args[:1])

		// Get current database
		repo := bpmutilsshared.GetRepository()
		if repo == "" {
			log.Fatal("Error: this command may only be run inside a BPM repository")
		}

		listPackagesFunc(repo)
	case "check-versions", "v":
		// Setup flags and help
		flagset := flag.NewFlagSet("check-versions", flag.ExitOnError)
		flagset.BoolP("verbose", "v", false, "Show additional information about the current operation")
		flagset.BoolP("force", "f", false, "Force current operation to bypass certain conditions")
		flagset.BoolP("apply", "a", false, "Apply new versions to packages")
		setupFlagsAndHelp(flagset, fmt.Sprintf("bpm-repo %s <options>", subcommand), "Manage BPM repositories and databases", os.Args[2:])
		currentFlagSet = flagset

		// Get current database
		repo := bpmutilsshared.GetRepository()
		if repo == "" {
			log.Fatal("Error: this command may only be run inside a BPM repository")
		}

		checkVersionsFunc(repo)
	default:
		log.Println("Error: unknown subcommand")
		listSubcommands()
		os.Exit(1)
	}
}

func createRepository(name, description string) {
	err := os.Mkdir(name, 0755)
	if err != nil {
		log.Fatalf("Error: could not create directory: %s", err)
	}

	contents := fmt.Sprintf("name: %s\ndescription: %s\n", name, description)
	err = os.WriteFile(path.Join(name, "bpm-repo.conf"), []byte(contents), 0644)
	if err != nil {
		log.Fatalf("Error: could not write to file: %s", err)
	}

	err = os.Mkdir(path.Join(name, "source"), 0755)
	if err != nil {
		log.Fatalf("Error: could not create directory: %s", err)
	}

	fmt.Println("Repository created successfully!")
}

func checkVersionsFunc(repo string) {
	// Get flags
	verbose, _ := currentFlagSet.GetBool("verbose")
	force, _ := currentFlagSet.GetBool("force")
	apply, _ := currentFlagSet.GetBool("apply")

	// Read environment files
	err := readEnvFile(repo)
	if err != nil {
		log.Fatalf("Error: could not read environment file: %s", err)
	}

	// Read version cache
	type CachedVersionEntry struct {
		LatestVersion string `yaml:"latest_version"`
		Timestamp     int64  `yaml:"timestamp"`
	}
	cachedVersions := make(map[string]CachedVersionEntry)
	data, err := os.ReadFile(path.Join(repo, ".version-cache"))
	if err == nil {
		err := yaml.Unmarshal(data, &cachedVersions)
		if err != nil {
			cachedVersions = make(map[string]CachedVersionEntry)
		}
	}

	directories := make([]string, 0)
	if currentFlagSet.NArg() > 0 {
		for _, dir := range currentFlagSet.Args() {
			if _, err := os.Stat(path.Join(repo, "source", dir, "pkg.info")); err != nil {
				log.Fatalf("Error: could not find pkg.info file in directory (%s): %s", dir, err)
			}
			directories = append(directories, path.Join(repo, "source", dir))
		}
	} else {
		// Loop through each directory with a 'pkg.info' file
		err := filepath.Walk(path.Join(repo, "source"), func(path string, info fs.FileInfo, err error) error {
			if filepath.Base(path) == "pkg.info" {
				directories = append(directories, filepath.Dir(path))
			}
			return nil
		})
		if err != nil {
			log.Fatalf("Error: could not loop through all packages: %s", err)
		}
	}

	pkgsWithoutScript := make([]string, 0)
	pkgsIgnored := make([]string, 0)
	pkgsWithError := make(map[string]error)
	pkgsWithUpdates := make(map[string]struct {
		OldVersion string
		NewVersion string
	}, 0)
	pkgsUpToDate := 0
	for _, dir := range directories {
		pkgInfo, err := bpmutilsshared.ReadPacakgeInfoFromFile(path.Join(dir, "pkg.info"))
		if err != nil {
			log.Fatalf("Could not read package info: %s", err)
		}

		if verbose {
			fmt.Printf("Checking version for package (%s)...\n", pkgInfo.Name)
		}

		// Check cached latest version
		latestVersion := ""
		if cachedVersion, ok := cachedVersions[pkgInfo.Name]; ok && !force && time.Since(time.UnixMilli(cachedVersion.Timestamp)).Milliseconds() < 604800000 {
			latestVersion = cachedVersion.LatestVersion
		} else {
			// Check whether check-version.sh script exists
			if _, err := os.Stat(path.Join(dir, "check-version.sh")); err != nil {
				pkgsWithoutScript = append(pkgsWithoutScript, pkgInfo.Name)
				continue
			}

			// Execute check-version.sh script
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", "-e", path.Join(dir, "check-version.sh"))
			cmd.Environ()
			output, err := cmd.Output()
			if err != nil {
				pkgsWithError[pkgInfo.Name] = err
				continue
			}
			latestVersion = strings.TrimSpace(string(output))

			// Check if package should be ignored
			if latestVersion == "ignore" {
				pkgsIgnored = append(pkgsIgnored, pkgInfo.Name)

				// Remove cached status
				delete(cachedVersions, pkgInfo.Name)

				continue
			}

			// Ensure latest version is valid
			if latestVersion == "" || latestVersion == "null" {
				pkgsWithError[pkgInfo.Name] = fmt.Errorf("invalid version number \"%s\"", latestVersion)
				continue
			}

			// Cache latest version
			cachedVersions[pkgInfo.Name] = CachedVersionEntry{
				LatestVersion: latestVersion,
				Timestamp:     time.Now().UnixMilli(),
			}
		}

		// Compare versions
		if pkgInfo.Version != latestVersion {
			pkgsWithUpdates[pkgInfo.Name] = struct {
				OldVersion string
				NewVersion string
			}{
				OldVersion: pkgInfo.Version,
				NewVersion: latestVersion,
			}

			// Apply new version
			pkgInfo.Version = latestVersion
			pkgInfo.Revision = 1
			if apply {
				var data bytes.Buffer
				encoder := yaml.NewEncoder(&data)
				encoder.SetIndent(2)
				encoder.Encode(pkgInfo)

				// Write package information
				fmt.Println(path.Join(dir, "pkg.info"))
				err := os.WriteFile(path.Join(dir, "pkg.info"), data.Bytes(), 0644)
				if err != nil {
					log.Printf("Warning: could not write new version for package (%s) to file: %s", pkgInfo.Name, err)
				}

				// Generate source package
				cmd := exec.Command("bpm-package")
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					log.Printf("Warning: could not generate source pacakge (%s): %s", pkgInfo.Name, err)
				}
			}

			continue
		}

		pkgsUpToDate++
	}

	// Save cached versions to file
	data, err = yaml.Marshal(cachedVersions)
	if err == nil {
		err := os.WriteFile(path.Join(repo, ".version-cache"), data, 0644)
		if err != nil {
			log.Printf("Warning: could not write cached versions to file: %s", err)
		}
	}

	// Print updates
	keys := slices.Collect(maps.Keys(pkgsWithUpdates))
	sort.Strings(keys)
	for _, pkg := range keys {
		fmt.Printf("Update available for package (%s): %s -> %s\n", pkg, pkgsWithUpdates[pkg].OldVersion, pkgsWithUpdates[pkg].NewVersion)
	}

	if verbose {
		// Print ignored packages
		for _, pkg := range pkgsIgnored {
			log.Printf("Warning: package (%s) was ignored\n", pkg)
		}

		// Print packages without check-version.sh script
		for _, pkg := range pkgsWithoutScript {
			log.Printf("Warning: package (%s) has no check-version.sh script\n", pkg)
		}
	}

	// Print errors
	keys = slices.Collect(maps.Keys(pkgsWithError))
	sort.Strings(keys)
	for _, pkg := range keys {
		log.Printf("Error: check-version.sh script for package (%s) failed: %s", pkg, pkgsWithError[pkg])
	}

	// Print summary
	fmt.Println("----- Summary -----")
	fmt.Println("Available updates:", len(pkgsWithUpdates))
	fmt.Println("Up to date:", pkgsUpToDate)
	fmt.Println("Missing script:", len(pkgsWithoutScript))
	fmt.Println("Ignored: ", len(pkgsIgnored))
	fmt.Println("Errors:", len(pkgsWithError))
}

func listPackagesFunc(repo string) {
	// Read databases
	sourceDatabase, err := bpmutilsshared.ReadDatabase(path.Join(repo, "source/database.bpmdb"))
	if err != nil {
		return
	}

	binaryDatabase, err := bpmutilsshared.ReadDatabase(path.Join(repo, "binary/database.bpmdb"))
	if err != nil {
		binaryDatabase = &bpmutilsshared.BPMDatabase{}
	}

	// Sort database entries
	entriesSorted := slices.Collect(maps.Values(sourceDatabase.Entries))
	sort.Slice(entriesSorted, func(i, j int) bool {
		return entriesSorted[i].PackageInfo.Name < entriesSorted[j].PackageInfo.Name
	})

	// Loop through each entry
	for _, entry := range entriesSorted {
		if len(entry.PackageInfo.SplitPackages) > 0 {
			for _, splitPkg := range entry.PackageInfo.SplitPackages {
				// Handle split packages
				additionalText := ""
				if binaryEntry, ok := binaryDatabase.Entries[splitPkg.Name]; !ok {
					additionalText += "(Binary package missing) "
				} else if fmt.Sprintf("%s-%d", binaryEntry.PackageInfo.Version, binaryEntry.PackageInfo.Revision) != fmt.Sprintf("%s-%d", entry.PackageInfo.Version, entry.PackageInfo.Revision) {
					additionalText += "(Binary package version mismatch) "
				}
				fmt.Printf("%s %s-%d %s\n", splitPkg.Name, entry.PackageInfo.Version, entry.PackageInfo.Revision, additionalText)
			}
		} else {
			additionalText := ""
			if binaryEntry, ok := binaryDatabase.Entries[entry.PackageInfo.Name]; !ok {
				additionalText += "(Binary package missing) "
			} else if fmt.Sprintf("%s-%d", binaryEntry.PackageInfo.Version, binaryEntry.PackageInfo.Revision) != fmt.Sprintf("%s-%d", entry.PackageInfo.Version, entry.PackageInfo.Revision) {
				additionalText += "(Binary package version mismatch) "
			}
			fmt.Printf("%s %s-%d %s\n", entry.PackageInfo.Name, entry.PackageInfo.Version, entry.PackageInfo.Revision, additionalText)
		}
	}
}

func readEnvFile(repo string) error {
	data, err := os.ReadFile(path.Join(repo, ".env"))
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}

		splitLine := strings.Split(line, "=")
		if len(splitLine) != 2 {
			return fmt.Errorf("invalid format")
		}

		os.Setenv(splitLine[0], splitLine[1])
	}

	return nil
}

func listSubcommands() {
	fmt.Println("Usage: bpm-repo <subcommand> <options>")
	fmt.Println("Description: Manage BPM repositories and databases")
	fmt.Println("Subcommands:")
	fmt.Println("  c, create-repo      Create a new BPM repository")
	fmt.Println("  u, update-db        Update update source and binary databases in current repositor")
	fmt.Println("  v, check-versions   Manage BPM repositories and databases")
	fmt.Println("  l, list             List packages")
}

func setupFlagsAndHelp(flagset *flag.FlagSet, usage, desc string, args []string) {
	flagset.Usage = func() {
		fmt.Println("Usage: " + usage)
		fmt.Println("Description: " + desc)
		fmt.Println("Options:")
		flagset.PrintDefaults()
	}
	flagset.Parse(args)
}
