package database

import (
	"icapeg/logging"
	"os"
)

var DBFile *os.File

func NewDatabase(fileDir string) (*os.File, error) {

	DBFile, err := os.Open(fileDir)

	if err != nil {
		logging.Logger.Info("cannot open file")
		return nil, err
	}
	return DBFile, nil
}
