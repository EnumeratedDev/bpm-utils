package bpm_utils_shared

import (
	"os"
	"path"
	"sort"
)

func GetRepository() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for dir != "/" {
		if _, err := os.Stat(path.Join(dir, "bpm-repo.conf")); err == nil {
			return dir
		}
		dir = path.Dir(dir)
	}

	if dir == "/" {
		return ""
	}
	return dir
}

func ReadRepositoryRecipes(repository string) []PackageInfo {
	// Read package recipes
	recipeDirs, err := os.ReadDir(path.Join(repository, "recipes"))
	if err != nil {
		return nil
	}

	pkgs := make([]PackageInfo, 0)
	for _, dir := range recipeDirs {
		pkgInfoPath := path.Join(repository, "recipes", dir.Name(), "info.yml")
		if _, err := os.Stat(pkgInfoPath); err != nil {
			continue
		}

		pkgInfo, err := ReadPacakgeInfoFromFile(pkgInfoPath)
		if err == nil {
			pkgs = append(pkgs, *pkgInfo)
		}
	}

	// Sort package recipes
	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].Name < pkgs[j].Name
	})

	return pkgs
}
