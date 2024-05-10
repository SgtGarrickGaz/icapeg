package database

import (
	"os"
)

var DBFile *os.File

func NewDatabase(fileDir string) (*os.File, error) {

	DBFile, err := os.Open(fileDir)

	if err != nil {
		return nil, err
	}
	return DBFile, nil
}
