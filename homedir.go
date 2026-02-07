package sshw

import (
	"os/user"
	"path/filepath"
)

// expandHome expands a leading ~ in path to the user's home directory.
func expandHome(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}
	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return path, nil
	}
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, path[1:]), nil
}
