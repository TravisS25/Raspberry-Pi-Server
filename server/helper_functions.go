package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// getRoot gets the absolute current working directory
func getRoot() string {
	root, err := os.Getwd()

	if err != nil {
		log.Fatal(err)
	}

	return root
}

func initDirectories() {
	_, err := ioutil.ReadDir("csv/sets")

	if err != nil {
		os.MkdirAll("csv/sets", os.ModePerm)
	}
}

func initDeviceCenter() {
	deviceCenter = &devices{
		DeviceRecording: make(map[string]bool),
		DeviceSet:       make(map[string]string),
		NewDeviceSet:    make(map[string]bool),
		DeviceTime:      make(map[string]time.Time),
	}
	setDirectoryPath := filepath.Join("csv", "sets")
	err := os.MkdirAll(setDirectoryPath, os.ModePerm)

	if err != nil {
		log.Fatal(err)
	}

	files, err := ioutil.ReadDir("csv")

	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if !file.IsDir() && file.Name() != "logfile.csv" {
			fileName := strings.Split(file.Name(), ".")[0]
			deviceCenter.DeviceTime[fileName] = time.Now()
			deviceCenter.DeviceRecording[fileName] = true
			deviceCenter.NewDeviceSet[fileName] = false
			err := os.MkdirAll(filepath.Join(setDirectoryPath, fileName), os.ModePerm)

			if err != nil {
				log.Fatal(err)
			}
		}
	}

	setDirectories, err := ioutil.ReadDir(setDirectoryPath)

	if err != nil {
		log.Fatal(err)
	}

	for _, directory := range setDirectories {
		setFiles, err := ioutil.ReadDir(filepath.Join(setDirectoryPath, directory.Name()))

		if err != nil {
			log.Fatal(err)
		}

		if len(setFiles) == 0 {
			deviceCenter.DeviceSet[directory.Name()] = "0"
		} else {
			lastSetFile := setFiles[len(setFiles)-1]
			deviceCenter.DeviceSet[directory.Name()] = strings.Split(lastSetFile.Name(), ".")[0]
		}
	}
}

func sendPayload(w http.ResponseWriter, payload interface{}) {
	jsonString, err := json.Marshal(payload)

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonString)
	}
}

// handlePostRequests makes sure that incoming requests are of method "POST" and that
// they have the write password.  This is used for api end points that usually
// changes files
func handlePostRequests(w http.ResponseWriter, r *http.Request) (err error) {
	r.ParseForm()
	var message string

	if r.Method != "POST" {
		message = "Request method is not post"
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(message))
		return errors.New(message)
	}

	password := r.Form.Get("password")

	if password != "test" {
		message = "Wrong Password"
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(message))
		return errors.New(message)
	}

	return nil
}
