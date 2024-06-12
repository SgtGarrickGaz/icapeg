package database

import (
	"icapeg/logging"
	"os"
)

var DBFile *os.File

func NewDatabase(fileDir string) (*os.File, error) {

	DBFile, err := os.OpenFile(fileDir, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)

	if err != nil {
		logging.Logger.Info("cannot open file")
		return nil, err
	}
	return DBFile, nil
}
