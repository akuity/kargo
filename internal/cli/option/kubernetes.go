package option

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
)

// TODO: Test this
func ReadManifests(filenames ...string) ([]byte, error) {
	var allBytes [][]byte
	for _, filename := range filenames {
		var err error
		if filename, err = filepath.Abs(filename); err != nil {
			return nil, err
		}
		fileInfo, err := os.Stat(filename)
		if err != nil {
			return nil, err
		}
		var fileBytes []byte
		if fileInfo.IsDir() {
			dirEntries, err := os.ReadDir(filename)
			if err != nil {
				return nil, err
			}
			dirFiles := make([]string, 0, len(dirEntries))
			for _, dirEntry := range dirEntries {
				dirFiles = append(dirFiles, path.Join(filename, dirEntry.Name()))
			}
			if fileBytes, err = ReadManifests(dirFiles...); err != nil {
				return nil, err
			}
		} else {
			// Check file extension
			ext := filepath.Ext(filename)
			if ext != ".yaml" && ext != ".yml" {
				continue
			}
			if fileBytes, err = os.ReadFile(filename); err != nil {
				return nil, err
			}
		}
		if len(fileBytes) > 0 {
			if fileBytes[len(fileBytes)-1] == '\n' {
				fileBytes = fileBytes[:len(fileBytes)-1]
			}
			allBytes = append(allBytes, fileBytes)
		}
	}
	return bytes.Join(allBytes, []byte("\n---\n")), nil
}
