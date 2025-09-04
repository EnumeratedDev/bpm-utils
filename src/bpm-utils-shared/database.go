package bpm_utils_shared

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type BPMDatabase struct {
	DatabaseVersion int                         `yaml:"database_version"`
	Entries         map[string]BPMDatabaseEntry `yaml:"entries"`
}

type BPMDatabaseEntry struct {
	PackageInfo   *PackageInfo `yaml:"info"`
	Filepath      string       `yaml:"filepath"`
	DownloadSize  int64        `yaml:"download_size"`
	InstalledSize int64        `yaml:"installed_size"`
}

func ReadDatabase(path string) (*BPMDatabase, error) {
	// Read database file
	output, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	database := &BPMDatabase{}

	// Unmarshal yaml
	err = yaml.Unmarshal(output, &database)
	if err != nil {
		return nil, err
	}

	return database, nil
}

func GenerateDatabase(path string) error {
	database := BPMDatabase{
		DatabaseVersion: 2,
		Entries:         make(map[string]BPMDatabaseEntry),
	}

	err := filepath.Walk(path, func(packagePath string, info fs.FileInfo, err error) error {
		if !strings.HasSuffix(packagePath, ".bpm") {
			return nil
		}

		// Initialize database entry
		entry := BPMDatabaseEntry{}
		entry.DownloadSize = info.Size()
		entry.Filepath, err = filepath.Rel(path, packagePath)
		if err != nil {
			return err
		}

		// Get package installed size
		cmd := exec.Command("tar", "-t", "-v", "-f", packagePath, "files.tar.gz")
		output, err := cmd.Output()
		if err == nil {
			entry.InstalledSize, err = strconv.ParseInt(strings.Fields(string(output))[2], 10, 64)
			if err != nil {
				return err
			}
		} else if err.(*exec.ExitError).ExitCode() == 2 {
			entry.InstalledSize = 0
		} else {
			return err
		}

		// Get package info
		entry.PackageInfo, err = ReadPacakgeInfo(packagePath)
		if err != nil {
			return err
		}

		// Add entry to database
		if _, ok := database.Entries[entry.PackageInfo.Name]; ok {
			return fmt.Errorf("package (%s) has already been added to the database", entry.PackageInfo.Name)
		}
		database.Entries[entry.PackageInfo.Name] = entry

		return nil
	})
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(&database)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(path, "database.bpmdb"), data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func UpdateDatabases(repo string) {
	if _, err := os.Stat(path.Join(repo, "source")); err == nil {
		err = GenerateDatabase(path.Join(repo, "source"))
		if err != nil {
			log.Fatalf("Error: could not generate source directory database: %s", err)
		}
		fmt.Println("Source directory database was generated successfully!")
	}

	if _, err := os.Stat(path.Join(repo, "binary")); err == nil {
		err = GenerateDatabase(path.Join(repo, "binary"))
		if err != nil {
			log.Fatalf("Error: could not generate binary directory database: %s", err)
		}
		fmt.Println("Binary directory database was generated successfully!")
	}
}
