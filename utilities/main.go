package utilities

import (
	"os"
	"path/filepath"
	"strings"
)

func Keys[T comparable, K any](input map[T]K) []T {
	result := make([]T, len(input))
	i := 0
	for k, _ := range input {
		result[i] = k
		i++
	}

	return result
}

func AbsolutePath(targetPath string) (string, error) {
	if strings.Contains(targetPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		targetPath = strings.ReplaceAll(targetPath, "~", home)
	}
	return filepath.Abs(targetPath)
}
