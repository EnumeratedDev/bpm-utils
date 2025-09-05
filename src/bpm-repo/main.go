package main

import (
	bpmutilsshared "bpm-utils-shared"
	"bufio"
	"fmt"
	"log"
	"maps"
	"os"
	"path"
	"slices"
	"sort"
	"strings"

	flag "github.com/spf13/pflag"
)

var createRepo = flag.BoolP("create", "c", false, "Create a new BPM repository")
var updateDatabases = flag.BoolP("update-databases", "u", false, "Update update source and binary databases in current repository")
var listPackages = flag.BoolP("list", "l", false, "List packages")

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
	} else if *listPackages {
		// Get current database
		repo := bpmutilsshared.GetRepository()
		if repo == "" {
			log.Fatal("Error: this command may only be run inside a BPM repository")
		}

		listPacakges(repo)
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

func listPacakges(repo string) {
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
