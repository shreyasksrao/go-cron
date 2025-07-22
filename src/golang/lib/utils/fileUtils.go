package utils

import (
	"encoding/json"
	"os"

	"github.com/shreyasksrao/jobmanager/lib/core"
)

func CreateDirIfNotExist(log core.Logger, dirName string) (created bool, err error) {
	log.Infof("Creating the directory - %v", dirName)
	log.Infof("Checking the path existence for the path - %v", dirName)
	_, _err := os.Stat(dirName)
	if os.IsNotExist(_err) {
		log.Infof("'%v' this directory not exists. Creating it...", dirName)
		err = os.MkdirAll(dirName, os.ModePerm)
		if err != nil {
			log.Errorf("Error occurred while creating the directory. \nError: %v", err)
			return
		}
		log.Infof("Successfully created the directory - %v", dirName)
		return
	} else {
		log.Infof("Directory - '%v' already exists.", dirName)
		return true, nil
	}
}

func CheckFileExistance(log core.Logger, filename string) (exists bool) {
	log.Infof("Checking if file exists or not for the filename - %v", filename)
	_, err := os.Stat(filename)
	if err != nil {
		log.Infof("File - '%v' doesn't exist.", filename)
		return false
	}
	log.Infof("File - '%v' exists.", filename)
	return true
}

func CreateEmptyJsonFileIfNotExist(log core.Logger, filePath string) (err error) {
	jobFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			log.Infof("JSON file %v already exists.", filePath)
			return nil
		}
		log.Errorf("Failed to create the JSON file - %v. Error: %v", filePath, err)
		return
	}
	emptyObject := map[string]interface{}{}
	// Write the empty JSON object to the file
	encoder := json.NewEncoder(jobFile)
	encoder.SetIndent("", "  ") // Pretty print with indentation
	if err = encoder.Encode(emptyObject); err != nil {
		log.Errorf("Error encoding to JSON:", err)
		return
	}
	return
}
