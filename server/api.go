package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// mainView displays the main html page with charts
func mainView(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "index.html", deviceCenter)
}

func newSetHandler(w http.ResponseWriter, r *http.Request) {
	err := handlePostRequests(w, r)

	if err != nil {
		return
	}

	devices := r.Form.Get("new-set")

	if devices == "" {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Must select at least one device to start new set"))
		return
	}

	mu.Lock()
	defer mu.Unlock()
	var newFile *os.File
	setsDirectory := filepath.Join("csv", "sets")
	deviceSet := make(map[string]string)

	for _, deviceName := range r.Form["new-set"] {
		currentCSVFilePath := filepath.Join("csv", deviceName+".csv")
		deviceSetDirectory := filepath.Join(setsDirectory, deviceName)
		fileInfoArray, err := ioutil.ReadDir(deviceSetDirectory)

		if err != nil {
			log.Fatal(err)
		}

		currentCSVFile, err := os.Open(currentCSVFilePath)

		if err != nil {
			log.Fatal(err)
		}

		if len(fileInfoArray) == 0 {
			newFile, err = os.OpenFile(filepath.Join(deviceSetDirectory, "1.csv"), os.O_WRONLY|os.O_CREATE, os.ModePerm)

			if err != nil {
				log.Fatal(err)
			}

			deviceSet[deviceName] = "1"
			io.Copy(newFile, currentCSVFile)
		} else {
			lastFileInfo := fileInfoArray[len(fileInfoArray)-1]
			newFileName, err := strconv.Atoi(strings.Split(lastFileInfo.Name(), ".")[0])

			if err != nil {
				log.Fatal(err)
			}

			newFileName++
			stringFileName := strconv.Itoa(newFileName)
			newFile, err := os.OpenFile(filepath.Join(deviceSetDirectory, stringFileName+".csv"), os.O_WRONLY|os.O_CREATE, os.ModePerm)

			if err != nil {
				log.Fatal(err)
			}

			deviceSet[deviceName] = stringFileName
			io.Copy(newFile, currentCSVFile)
		}

		newFile.Close()
		currentCSVFile.Close()
		os.Remove(currentCSVFilePath)
		_, err = os.Create(currentCSVFilePath)

		if err != nil {
			log.Fatal(err)
		}

		deviceCenter.Lock()
		deviceCenter.NewDeviceSet[deviceName] = true
		deviceCenter.Unlock()
	}

	sendPayload(w, deviceSet)
}

func reloadCSVHandler(w http.ResponseWriter, r *http.Request) {
	file, handler, err := r.FormFile("uploadFile")
	defer file.Close()

	if err != nil {
		log.Fatal(err)
	}

	err = handlePostRequests(w, r)

	if err != nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()
	pathToFile := filepath.Join("csv", handler.Filename)
	os.Remove(pathToFile)
	f, err := os.OpenFile(pathToFile, os.O_WRONLY|os.O_CREATE, 0666)

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()
	io.Copy(f, file)
}

// recordingHandler is an api endpoint that will get a list of device
// names from html page and will either start or stop recording
// based on device names given and flag to start or stop recording
func recordModeHandler(w http.ResponseWriter, r *http.Request) {
	err := handlePostRequests(w, r)

	if err != nil {
		return
	}

	record := r.Form.Get("record-mode")
	devices := r.Form.Get("record-device")

	if devices == "" {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Must select at least one device to change mode for"))
		return
	}

	if record == "" {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Must choose whether to record or not"))
		return
	}

	deviceCenter.Lock()

	if deviceNames, ok := r.Form["record-device"]; ok {
		for _, deviceName := range deviceNames {
			if _, ok := deviceCenter.DeviceRecording[deviceName]; ok {
				if record == "true" {
					deviceCenter.DeviceRecording[deviceName] = true
				} else {
					deviceCenter.DeviceRecording[deviceName] = false
				}
			}
		}
	}

	deviceCenter.Unlock()
	sendPayload(w, deviceCenter.DeviceRecording)
}

// recordingStatusHandler is an api endpoint that checks the statuses of
// all devices and if a device hasn't been heard from for certain
// amount of time, will add device name to list and give warning on webpage
func updateStatusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("record status")
	now := time.Now()
	devicesNotHeardFrom := make(map[string]time.Time)

	deviceCenter.RLock()
	// If device hasn't been recorded for more than 5 seconds, add to list
	for deviceName, deviceTime := range deviceCenter.DeviceTime {
		if deviceTime.Before(now.Add(timeOut*time.Second)) && deviceCenter.DeviceRecording[deviceName] {
			devicesNotHeardFrom[deviceName] = deviceTime
		}
	}

	deviceCenter.RUnlock()
	sendPayload(w, devicesNotHeardFrom)
	return
}

// deviceRecordingStatusHandler is an api end point that checks if passed
// device name is allowed to record or not
// This api will be pinged by a device that has stopped recording and will
// continue to check if the device is allowed to record again
func deviceStatusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("device record")
	message := ""
	deviceName := r.URL.Query().Get("deviceName")

	if deviceName == "" {
		message += "Must give device name,"
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte(message))
		return
	}

	deviceCenter.RLock()
	record, recordOK := deviceCenter.DeviceRecording[deviceName]
	newSet, newSetOK := deviceCenter.NewDeviceSet[deviceName]
	deviceCenter.RUnlock()

	if recordOK || newSetOK {
		w.WriteHeader(http.StatusOK)

		if recordOK {
			if record {
				message += "Record,"
			} else {
				message += "Stop Recording,"
			}
		}

		if newSetOK {
			if newSet {
				message += "New Set"
			} else {
				message += "Continue Set"
			}
		}
	} else {
		message += "Device name does not exist"
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(message))
	}

	return
}

// sensorHandler is an api endpoint that receives time stamp info from our devices
// and adds them to their own device log file and an overall log file
func sensorHandler(w http.ResponseWriter, r *http.Request) {
	err := handlePostRequests(w, r)

	if err != nil {
		return
	}

	fmt.Println("Sensor reached")
	var deviceFile *os.File
	var message string
	timeStamp := r.Form.Get("timeStamp")
	deviceName := strings.Split(timeStamp, ",")[0]
	fmt.Println(deviceCenter.DeviceRecording)

	deviceCenter.RLock()
	deviceCenter.DeviceTime[deviceName] = time.Now()
	recording, recordingOK := deviceCenter.DeviceRecording[deviceName]
	newSet, newSetOK := deviceCenter.NewDeviceSet[deviceName]
	deviceCenter.RUnlock()

	// If either map contains the device name, begin forming message
	// Else add new device name to deviceCenter variable
	if recordingOK || newSetOK {
		w.WriteHeader(http.StatusOK)

		if recordingOK {
			// If device is issued to stop recording, send message to device
			// to stop recording
			// Else begin/continue recording
			if !recording {
				message += "Stop Recording,"
			} else {
				message += "Record,"
			}
		}

		if newSetOK {
			// If device is issued to start new set, send message to device
			// to start new set which the device will delete local file
			// and start new
			// Else continue current set
			if newSet {
				message += "New Set"
				deviceCenter.Lock()
				deviceCenter.NewDeviceSet[deviceName] = false
				deviceCenter.Unlock()
			} else {
				message += "Continue Set"
			}
		}

		// Write message to device
		w.Write([]byte(message))

	} else {
		deviceCenter.Lock()
		deviceCenter.DeviceRecording[deviceName] = true
		deviceCenter.DeviceSet[deviceName] = "0"
		deviceCenter.NewDeviceSet[deviceName] = false
		deviceCenter.Unlock()

		// If current request is from new device, create directory with device
		// name under the sets directory
		err := os.MkdirAll(filepath.Join("csv", "sets", deviceName), os.ModePerm)

		if err != nil {
			log.Fatal(err)
		}
	}

	deviceFilePath := filepath.Join("csv", deviceName+".csv")
	_, deviceErr := os.Stat(deviceFilePath)
	// _, logErr := os.Stat(logFilePath)

	mu.Lock()
	defer mu.Unlock()

	// If either log file for device or overall do not exists, create them and write
	// time stamp to them
	// Else append time stamp to file
	if deviceErr != nil {
		deviceFile, _ = os.Create(deviceFilePath)
		deviceFile.WriteString(timeStamp)
		// deviceRecording[deviceName] = true
	} else {
		deviceFile, err = os.OpenFile(deviceFilePath, os.O_APPEND|os.O_WRONLY, 0666)

		if err != nil {
			log.Println(err)
		} else {
			deviceFile.WriteString(timeStamp)
		}
	}

	defer deviceFile.Close()

	// if logErr != nil {
	// 	logFile, _ = os.Create(logFilePath)
	// 	logFile.WriteString(timeStamp)
	// } else {
	// 	logFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_WRONLY, 0666)

	// 	if err != nil {
	// 		log.Println(err)
	// 	} else {
	// 		logFile.WriteString(timeStamp)
	// 	}
	// }

	// defer logFile.Close()
}

// // updateChartHandler is an api point that will read our overall log file,
// // calculate the total amount of motion based on the time measurement passed
// // and return
// func updateChartHandler(w http.ResponseWriter, r *http.Request) {
// 	r.ParseForm()
// 	// var startingTime time.Time
// 	timeMeasure := r.Form.Get("timeMeasure")
// 	now := time.Now()
// 	payload := make(map[string]map[int]int)

// 	mu.RLock()
// 	defer mu.Unlock()
// 	logFile, err := os.Open(logFilePath)

// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	defer logFile.Close()
// 	reader := bufio.NewReader(logFile)
// 	hourPayload := func(dateTime time.Time, payload map[string]map[int]int) {
// 		startingTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
// 		for i := 0; i < 60; i = i + 10 {
// 			if i == 0 {
// 				previousHour := startingTime.Add(-10 * time.Minute)

// 				if dateTime.After(previousHour) && dateTime.Before(startingTime) {
// 					payload["hour"][i]++
// 				}
// 			} else if dateTime.After(startingTime.Add(-10*time.Minute)) && dateTime.Before(startingTime.Add(time.Duration(i)*time.Minute)) {
// 				payload["hour"][i]++
// 				break
// 			}
// 		}
// 	}
// 	dayPayload := func(dateTime time.Time, payload map[string]map[int]int) {
// 		startingTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
// 		for i := 0; i < 24; i++ {
// 			if i == 0 {
// 				previousDay := startingTime.Add(-1 * time.Hour)

// 				if dateTime.After(previousDay) && dateTime.Before(startingTime) {
// 					payload["day"][i]++
// 				}
// 			} else if dateTime.After(startingTime.Add(-1*time.Hour)) && dateTime.Before(startingTime.Add(time.Duration(i)*time.Hour)) {
// 				payload["day"][i]++
// 				break
// 			}
// 		}
// 	}

// 	for {
// 		line, err := reader.ReadString('\n')
// 		timeStampArray := strings.Split(line, ",")
// 		dateTime, timeErr := time.Parse("06/01/2006 11:20:10", timeStampArray[1]+" "+timeStampArray[2])

// 		if timeErr != nil {
// 			dateTime = time.Now()
// 		}

// 		if timeStampArray[3] == "T" {
// 			switch timeMeasure {
// 			case "hour":
// 				hourPayload(dateTime, payload)
// 			case "day":
// 				dayPayload(dateTime, payload)
// 			case "all":
// 				hourPayload(dateTime, payload)
// 				dayPayload(dateTime, payload)
// 			default:
// 				dayPayload(dateTime, payload)
// 			}
// 		}

// 		if err != nil {
// 			break
// 		}
// 	}

// 	sendPayload(w, payload)
// 	return
// }

// updateChartHandler is an api point that will read our overall log file,
// calculate the total amount of motion based on the time measurement passed
// and return
func updateChartHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	timeMeasure := r.Form.Get("timeMeasure")
	now := time.Now()

	hourPayload := func(dateTime time.Time, payload *chart) {
		payload.TimeMeasure = "hour"
		startingTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
		for i := 0; i < 60; i = i + 10 {
			if i == 0 {
				previousHour := startingTime.Add(-10 * time.Minute)

				if dateTime.After(previousHour) && dateTime.Before(startingTime) {
					payload.Axises[i]++
				}
			} else if dateTime.After(startingTime.Add(-10*time.Minute)) && dateTime.Before(startingTime.Add(time.Duration(i)*time.Minute)) {
				payload.Axises[i]++
				break
			}
		}
	}
	dayPayload := func(dateTime time.Time, payload *chart) {
		payload.TimeMeasure = "day"
		startingTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		for i := 0; i < 24; i++ {
			if i == 0 {
				previousDay := startingTime.Add(-1 * time.Hour)

				if dateTime.After(previousDay) && dateTime.Before(startingTime) {
					payload.Axises[i]++
				}
			} else if dateTime.After(startingTime.Add(-1*time.Hour)) && dateTime.Before(startingTime.Add(time.Duration(i)*time.Hour)) {
				payload.Axises[i]++
				break
			}
		}
	}

	mu.RLock()
	defer mu.Unlock()

	fileInfoArray, err := ioutil.ReadDir("csv")

	if err != nil {
		log.Fatal(err)
	}

	chartArray := make([]*chart, len(fileInfoArray))

	for i, fileInfo := range fileInfoArray {
		if !fileInfo.IsDir() {
			chartArray[i].DeviceName = fileInfo.Name()
			csvFile := filepath.Join("csv", fileInfo.Name()+".csv")
			file, err := os.Open(csvFile)

			if err != nil {
				log.Fatal(err)
			}

			reader := bufio.NewReader(file)

			for {
				line, err := reader.ReadString('\n')
				timeStampArray := strings.Split(line, ",")
				dateTime, timeErr := time.Parse("06/01/2006 11:20:10", timeStampArray[1]+" "+timeStampArray[2])

				if timeErr != nil {
					dateTime = time.Now()
				}

				if timeStampArray[3] == "T" {
					switch timeMeasure {
					case "hour":
						hourPayload(dateTime, chartArray[i])
					case "day":
						dayPayload(dateTime, chartArray[i])
					case "all":
						hourPayload(dateTime, chartArray[i])
						dayPayload(dateTime, chartArray[i])
					default:
						dayPayload(dateTime, chartArray[i])
					}
				}

				if err != nil {
					break
				}
			}
		}
	}

	sendPayload(w, chartArray)
	return
}
