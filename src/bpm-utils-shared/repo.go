package bpm_utils_shared

import (
	"os"
	"path"
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
