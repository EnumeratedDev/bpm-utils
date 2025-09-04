package main

import (
	bpmutilsshared "bpm-utils-shared"
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	flag "github.com/spf13/pflag"
)

var createRepo = flag.BoolP("create", "c", false, "Create a new BPM repository")
var updateDatabases = flag.BoolP("update-databases", "u", false, "Update update source and binary databases in current repository")

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
