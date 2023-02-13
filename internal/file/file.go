package file

import "os"

// Exists returns a bool indicating if the specified file exists or not. It
// returns any errors that are encountered that are NOT an os.ErrNotExist error.
func Exists(filename string) (bool, error) {
	_, err := os.Stat(filename)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
