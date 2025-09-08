package main

import (
	bpmutilsshared "bpm-utils-shared"
	"bufio"
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

var createRepo = flag.BoolP("create", "c", false, "Create a new BPM repository")
var updateDatabases = flag.BoolP("update-databases", "u", false, "Update update source and binary databases in current repository")
var checkVersions = flag.BoolP("check-versions", "v", false, "Get the latest version of each package")
var listPackages = flag.BoolP("list", "l", false, "List packages")

var verbose = flag.Bool("verbose", false, "Show additional information about current operation")
var force = flag.BoolP("force", "f", false, "Force current operation to bypass certain conditions")

func main() {
	// Setup flags and help
	bpmutilsshared.SetupHelp("bpm-repo <options>", "Manage BPM repositories and databases")
	bpmutilsshared.SetupFlags()

	if *createRepo {
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
	} else if *updateDatabases {
		// Get current database
		repo := bpmutilsshared.GetRepository()
		if repo == "" {
			log.Fatal("Error: this command may only be run inside a BPM repository")
		}

		bpmutilsshared.UpdateDatabases(repo)
	} else if *checkVersions {
		// Get current database
		repo := bpmutilsshared.GetRepository()
		if repo == "" {
			log.Fatal("Error: this command may only be run inside a BPM repository")
		}

		checkVersionsFunc(repo)
	} else if *listPackages {
		// Get current database
		repo := bpmutilsshared.GetRepository()
		if repo == "" {
			log.Fatal("Error: this command may only be run inside a BPM repository")
		}

		listPackagesFunc(repo)
	} else {
		bpmutilsshared.ShowHelp()
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

	fmt.Println("Repository created successfully!")
}

func checkVersionsFunc(repo string) {
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
	if flag.NArg() > 0 {
		for _, dir := range flag.Args() {
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

		// Check cached latest version
		latestVersion := ""
		if cachedVersion, ok := cachedVersions[pkgInfo.Name]; ok && !*force && time.Since(time.UnixMilli(cachedVersion.Timestamp)).Milliseconds() < 604800000 {
			latestVersion = cachedVersion.LatestVersion
		} else {
			// Check whether check-version.sh script exists
			if _, err := os.Stat(path.Join(dir, "check-version.sh")); err != nil {
				pkgsWithoutScript = append(pkgsWithoutScript, pkgInfo.Name)
				continue
			}

			// Execute check-version.sh script
			cmd := exec.Command("bash", "-e", path.Join(dir, "check-version.sh"))
			cmd.Environ()
			output, err := cmd.Output()
			if err != nil {
				pkgsWithError[pkgInfo.Name] = err
				continue
			}
			latestVersion = strings.TrimSpace(string(output))

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

	if *verbose {
		// Print packages without check-version.sh script
		for _, pkg := range pkgsWithoutScript {
			log.Printf("Warning: package (%s) has no check-version.sh script\n", pkg)
		}

		// Print errors
		keys = slices.Collect(maps.Keys(pkgsWithError))
		sort.Strings(keys)
		for _, pkg := range keys {
			log.Printf("Error: check-version.sh script for package (%s) failed: %s", pkg, pkgsWithError[pkg])
		}
	}

	// Print summary
	fmt.Println("----- Summary -----")
	fmt.Println("Available updates:", len(pkgsWithUpdates))
	fmt.Println("Up to date:", pkgsUpToDate)
	fmt.Println("Missing script:", len(pkgsWithoutScript))
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
