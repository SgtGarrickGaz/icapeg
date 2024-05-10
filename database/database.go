package database

import "os"

func NewDatabase(fileDir string) (*os.File, error) {
	file, err := os.OpenFile(fileDir, os.O_RDONLY|os.O_CREATE, 0666)

	if err != nil {
		return os.Create(fileDir)
	}

	return file, nil
}
