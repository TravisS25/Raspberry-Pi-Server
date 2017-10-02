package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// mainView displays the main html page with charts
func mainView(w http.ResponseWriter, r *http.Request) {
	context := map[string]interface{}{
		"deviceCenter": deviceCenter,
	}
	tpl.ExecuteTemplate(w, "index.html", context)
}

func generateDeviceTarHandler(w http.ResponseWriter, r *http.Request) {
	err := handlePostRequests(w, r)

	if err != nil {
		return
	}

	deviceName := r.Form.Get("deviceName")

	if deviceName == "" {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Device name is required"))
		return
	}

	deviceDirectory := filepath.Join(setsDirectory, deviceName)
	_, err = os.Stat(deviceDirectory)

	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Device name does not exist"))
		return
	}

	fileInfoArray, err := ioutil.ReadDir(deviceDirectory)
	randomFileName := randomString(20)
	tempFilePath := filepath.Join("/tmp", randomFileName)
	mainFile, err := os.Create(tempFilePath + ".tar.gz")
	if err != nil {
		unableToRetrieveFiles(w, err)
		return
	}
	defer mainFile.Close()
	// set up the gzip writer
	gw := gzip.NewWriter(mainFile)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, fileInfo := range fileInfoArray {
		fullPath := filepath.Join(deviceDirectory, fileInfo.Name())
		file, err := os.Open(fullPath)

		if err == nil {
			hdr := &tar.Header{
				Name: fileInfo.Name(),
				Mode: 0666,
				Size: fileInfo.Size(),
			}

			if err := tw.WriteHeader(hdr); err != nil {
				unableToRetrieveFiles(w, err)
				return
			}

			if _, err := io.Copy(tw, file); err != nil {
				unableToRetrieveFiles(w, err)
				return
			}
			file.Close()
		}
	}

	sendPayload(w, map[string]string{
		"file": randomFileName,
	})

	return
}

func generateAllDevicesTarHandler(w http.ResponseWriter, r *http.Request) {
	err := handlePostRequests(w, r)

	if err != nil {
		return
	}

	rootDirArray, err := ioutil.ReadDir(setsDirectory)
	checkError(err, "", true)
	randomFileName := randomString(20)
	tempFilePath := filepath.Join("/tmp", randomFileName)
	mainFile, err := os.Create(tempFilePath + ".tar.gz")
	checkError(err, "", true)
	defer mainFile.Close()
	// set up the gzip writer
	gw := gzip.NewWriter(mainFile)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, dirInfo := range rootDirArray {
		dirName := dirInfo.Name()
		dirPath := filepath.Join(setsDirectory, dirName)
		dirArray, err := ioutil.ReadDir(dirPath)

		if err != nil {
			unableToRetrieveFiles(w, err)
			return
		}

		for _, fileInfo := range dirArray {
			fileName := fileInfo.Name()
			filePath := filepath.Join(dirName, fileName)
			hdr := &tar.Header{
				Name: filePath,
				Mode: 0666,
				Size: fileInfo.Size(),
			}

			if err := tw.WriteHeader(hdr); err != nil {
				unableToRetrieveFiles(w, err)
				return
			}

			file, err := os.Open(filepath.Join(dirPath, fileName))

			if err != nil {
				unableToRetrieveFiles(w, err)
				return
			}

			if _, err := io.Copy(tw, file); err != nil {
				unableToRetrieveFiles(w, err)
				return
			}
		}
	}

	sendPayload(w, map[string]string{
		"file": randomFileName,
	})
}

func downloadTarHandler(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("fileName")

	if fileName == "" {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Device name is required"))
		return
	}

	filePath := filepath.Join("/tmp", fileName+".tar.gz")
	file, err := os.Open(filePath)

	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	io.Copy(w, file)
	os.Remove(filePath)

	return
}

// deviceCheckInHandler is an api endpoint that either adds new devices to our
// global deviceCenter variable or checks in a device that already exists
func deviceCheckInHandler(w http.ResponseWriter, r *http.Request) {
	err := handlePostRequests(w, r)

	if err != nil {
		return
	}

	var sqlStatement string
	deviceName := r.Form.Get("deviceName")
	dev, deviceExists := deviceCenter.Devices[deviceName]

	// If device already exists, update database
	// Else insert the new device into database with default values
	if deviceExists {
		if dev.IsCheckedIn {
			w.WriteHeader(http.StatusNotAcceptable)
			w.Write([]byte("Device already checked in"))
			return
		}

		sqlStatement = "UPDATE device SET latest_check_in_time=?, is_checked_in=1 WHERE name=?"
		now := time.Now().UTC()

		deviceCenter.Lock()
		deviceCenter.Devices[dev.Name].LatestCheckInTime = now.Format("2006-01-02 15:04:05")
		deviceCenter.Devices[dev.Name].IsCheckedIn = true
		deviceCenter.Unlock()

		err := execTXQuery(sqlStatement, now, deviceName)
		checkError(err, "Update query error", true)
	} else {
		sqlStatement =
			"INSERT INTO device (name, set_num, latest_check_in_time, is_new_set, is_recording, is_checked_in) " +
				"VALUES (?,?,?,?,?,?);"

		deviceCenter.Lock()
		now := time.Now().Format("2006-01-02 15:04:05")
		deviceCenter.Devices[deviceName] = &device{
			Name:              deviceName,
			SetNum:            1,
			LatestCheckInTime: now,
			IsNewSet:          false,
			IsRecording:       true,
			IsCheckedIn:       true,
		}
		deviceCenter.Unlock()
		err := execTXQuery(sqlStatement, deviceName, 1, now, 0, 1, 1)
		checkError(err, "", true)
	}

	// If current request is from new device, create directory with device
	// name under the sets directory
	err = os.MkdirAll(filepath.Join(setsDirectory, deviceName), 0666)
	checkError(err, "Can't make sets directory", true)
}

// newSetHandler is an api endpoint that signals that the server will start new
// csv files based on the device names passed
// The devices contained in list will reset their local csv file the next time
// they ping the sensorHandler api point
func newSetHandler(w http.ResponseWriter, r *http.Request) {
	err := handlePostRequests(w, r)

	if err != nil {
		return
	}

	devices := r.Form.Get("new-set")

	// If no device names were passed, return and write error message to writer
	if devices == "" {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Must select at least one device to start new set"))
		return
	}

	var newFile *os.File
	var message string
	deviceArray := make([]device, 0)
	// chartArray := make([]chartRow, 0)
	sqlUpdate := "UPDATE device SET is_new_set=1, set_num=?, latest_set_time=? WHERE name=?"

	for _, deviceName := range r.Form["new-set"] {
		deviceCenter.RLock()
		dev, deviceExists := deviceCenter.Devices[deviceName]
		deviceCenter.RUnlock()

		if deviceExists {
			if dev.IsRecording {
				message += deviceName + " is recording.  Can only start new set when " +
					"device is NOT recording <br /> "
				continue
			}
			if dev.IsNewSet {
				message += deviceName + " still hasn't reset to new set <br /> "
				continue
			}

			now := time.Now().Format("2006-01-02 15:04:05")
			mu.Lock()
			currentCSVFilePath := filepath.Join(csvDirectory, deviceName+".csv")
			deviceSetDirectory := filepath.Join(setsDirectory, deviceName)
			fileInfoArray, err := ioutil.ReadDir(deviceSetDirectory)
			checkError(err, "Couldn't read device directory", true)

			currentCSVFile, err := os.Open(currentCSVFilePath)
			checkError(err, "Couldn't open csv file", true)

			// If device directory has no files in it, then we insert the first
			// csv file in the set
			// Else calculate what the next file name should be.  Since file names
			// are just numbers, we just simply increment from the last file name
			if len(fileInfoArray) == 0 {
				newFile, err = os.OpenFile(filepath.Join(deviceSetDirectory, "1.csv"), os.O_WRONLY|os.O_CREATE, os.ModePerm)
				checkError(err, "", true)

				deviceArray = append(deviceArray, device{
					Name:          deviceName,
					SetNum:        1,
					LatestSetTime: now,
				})

				_, err := io.Copy(newFile, currentCSVFile)
				checkError(err, "", true)
			} else {
				lastFileInfo := fileInfoArray[len(fileInfoArray)-1]
				setNum, err := strconv.Atoi(strings.Split(lastFileInfo.Name(), ".")[0])
				checkError(err, "", true)

				setNum++
				stringFileName := strconv.Itoa(setNum)
				newFile, err := os.OpenFile(filepath.Join(deviceSetDirectory, stringFileName+".csv"), os.O_WRONLY|os.O_CREATE, 0666)
				checkError(err, "", true)

				deviceArray = append(deviceArray, device{
					Name:          deviceName,
					SetNum:        setNum,
					LatestSetTime: now,
				})
				_, err = io.Copy(newFile, currentCSVFile)
				checkError(err, "", true)
			}

			newFile.Close()
			currentCSVFile.Close()

			// Simply calling create to overwrite current file
			_, err = os.Create(currentCSVFilePath)
			checkError(err, "", true)
			mu.Unlock()

			dev.SetNum++
			err = execTXQuery(sqlUpdate, dev.SetNum, now, deviceName)
			checkError(err, "", true)

			deviceCenter.Lock()
			deviceCenter.Devices[deviceName].IsNewSet = true
			deviceCenter.Devices[deviceName].SetNum++
			deviceCenter.Devices[deviceName].LatestSetTime = now
			deviceCenter.Unlock()
		}
	}

	sendPayload(w, map[string]interface{}{
		"devices": devices,
		"message": message,
	})
}

func reloadCSVHandler(w http.ResponseWriter, r *http.Request) {
	file, handler, err := r.FormFile("uploadFile")

	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Improper multipart header sent"))
		return
	}

	defer file.Close()
	err = handlePostRequests(w, r)

	if err != nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()
	pathToFile := filepath.Join(csvDirectory, handler.Filename)
	os.Remove(pathToFile)
	f, err := os.OpenFile(pathToFile, os.O_WRONLY|os.O_CREATE, 0666)
	checkError(err, "", true)
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

	if record != "true" && record != "false" {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Must choose whether to record or not"))
		return
	}

	isRecording, _ := strconv.ParseBool(record)
	devicesRecordStatus := make(map[string]bool)
	sqlUpdate := "UPDATE device SET is_recording=? WHERE name=?"
	deviceCenter.Lock()

	for _, deviceName := range r.Form["record-device"] {
		device, deviceExists := deviceCenter.Devices[deviceName]

		if deviceExists {
			if (device.IsRecording && !isRecording) || (!device.IsRecording && isRecording) {
				err = execTXQuery(sqlUpdate, isRecording, deviceName)
				checkError(err, "", true)
				deviceCenter.Lock()
				deviceCenter.Devices[deviceName].IsRecording = true
				deviceCenter.Unlock()
			}

			devicesRecordStatus[deviceName] = isRecording
		}
	}

	sendPayload(w, devicesRecordStatus)
}

// updateStatusHandler is an api endpoint that checks the statuses of
// all devices and if a device hasn't been heard from for certain
// amount of time, will add device name to list and give warning on webpage
func updateStatusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("record status")
	devicesNotHeardFrom := make(map[string]time.Time)
	deviceCenter.RLock()

	for _, dev := range deviceCenter.Devices {
		if !dev.IsCheckedIn {
			latestCheckInTime, err := time.Parse("2006-01-02 15:04:05", dev.LatestCheckInTime)
			checkError(err, "Error parsing time", true)
			devicesNotHeardFrom[dev.Name] = latestCheckInTime
		}
	}

	deviceCenter.RUnlock()
	sendPayload(w, devicesNotHeardFrom)

	return
}

// deviceStatusHandler is an api end point that checks if passed
// device name is allowed to record or not or if the device needs
// to start a new set
// This api will be pinged by a device that has stopped recording and will
// continue to check if the device is allowed to record again or start new
// set while not recording
func deviceStatusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("device record")
	message := ""
	deviceName := r.URL.Query().Get("deviceName")

	// Return error message if no device name is sent
	if deviceName == "" {
		message += "Must give device name,"
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte(message))
		return
	}

	deviceCenter.RLock()
	_, deviceExists := deviceCenter.Devices[deviceName]
	deviceCenter.RUnlock()

	if deviceExists {
		w.WriteHeader(http.StatusOK)
		now := time.Now().Format("2006-01-02 15:04:05")
		timeUpdateQuery := "UPDATE device SET latest_check_in_time=? WHERE name=?;"
		err := execTXQuery(timeUpdateQuery, now, deviceName)
		checkError(err, "", true)

		deviceCenter.Lock()
		deviceCenter.Devices[deviceName].LatestCheckInTime = now
		deviceCenter.Unlock()
	} else {
		w.WriteHeader(http.StatusNotFound)
		message += "Device name does not exist"
	}

	w.Write([]byte(message))

	return
}

// sensorHandler is an api endpoint that receives time stamp info from our devices
// and adds them to their own device log file
func sensorHandler(w http.ResponseWriter, r *http.Request) {
	err := handlePostRequests(w, r)

	if err != nil {
		return
	}

	fmt.Println("Sensor reached")
	var deviceFile *os.File
	var message string
	timeStamp := r.Form.Get("timeStamp")
	timeStampArray := strings.Split(timeStamp, ",")

	if len(timeStampArray) != 4 {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Improper time stamp sent"))
		return
	}

	movement, err := strconv.ParseBool(strings.TrimRight(strings.Split(timeStamp, ",")[3], "\n"))

	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Movement must be either true or false"))
		return
	}

	_, err = time.ParseInLocation("2006-01-02 15:04:05", timeStampArray[1]+" "+timeStampArray[2], time.UTC)

	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Improper time sent"))
		return
	}

	deviceName := strings.Split(timeStamp, ",")[0]
	deviceCenter.RLock()
	dev, deviceExists := deviceCenter.Devices[deviceName]
	deviceCenter.RUnlock()

	if deviceExists {
		if dev.IsRecording {
			message += "Record,"
		} else {
			message += "Stop Record,"
		}

		sqlUpdate :=
			"UPDATE device " +
				"SET latest_check_in_time=?, is_recording=?, is_new_set=0 " +
				"WHERE name=?;"

		err = execTXQuery(
			sqlUpdate,
			deviceCenter.Devices[deviceName].LatestCheckInTime,
			dev.IsRecording,
			deviceName,
		)
		checkError(err, "", true)

		if dev.IsNewSet {
			deviceCenter.Lock()
			deviceCenter.Devices[deviceName].IsNewSet = false
			deviceCenter.Unlock()
		}

		deviceFilePath := filepath.Join(csvDirectory, deviceName+".csv")
		checkError(err, "Can't parse bool", true)
		_, deviceErr := os.Stat(deviceFilePath)

		if movement {
			mu.Lock()
			defer mu.Unlock()

			newTimeStamp := timeStampArray[1] + "," + timeStampArray[2] + " \n"

			// If log file for device does not exist, create and write time stamp to it
			// Else append time stamp to file
			if deviceErr != nil {
				deviceFile, _ = os.Create(deviceFilePath)
				deviceFile.WriteString(newTimeStamp)
			} else {
				deviceFile, err = os.OpenFile(deviceFilePath, os.O_APPEND|os.O_WRONLY, os.ModePerm)
				checkError(err, "Can't open file", true)
				deviceFile.WriteString(newTimeStamp)
			}

			defer deviceFile.Close()
		}
	} else {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Device does not exist"))
	}

	return
}

// updateChartHandler is an api point that will read our overall log file,
// calculate the total amount of motion based on the time measurement passed
// and return
func updateChartHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	timeMeasure := r.Form.Get("timeMeasure")
	now := time.Now().UTC()

	// Function for getting chart with hour info
	hourPayload := func(dateTime time.Time, payload *chart) {
		payload.TimeMeasure = "hour"
		increment := 5
		startingTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

		// Loop by increment variable which is the tick marks (in minutes) that will be used
		// for our chart
		for i := 0; i < 60; i = i + increment {
			// If we are on first iteration of loop, check increment of previous hour and see if
			// dateTime falls within that time period.  If it does, add to payLoad
			// Else check current hour within increment and if dateTime falls within, increment payLoad
			if i == 0 {
				previousHour := startingTime.Add(-time.Duration(increment) * time.Minute)

				if dateTime.After(previousHour) && dateTime.Before(startingTime) {
					payload.Axises[i]++
				}
			} else if dateTime.After(startingTime.Add(-time.Duration(increment)*time.Minute)) && dateTime.Before(startingTime.Add(time.Duration(i)*time.Minute)) {
				payload.Axises[i]++
				break
			}
		}
	}

	// Function for getting chart with 24 hour info
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

	fileInfoArray, err := ioutil.ReadDir(csvDirectory)
	checkError(err, "Can't read files in directory", true)
	chartArray := make([]*chart, len(fileInfoArray))
	mu.RLock()

	for i, fileInfo := range fileInfoArray {
		if !fileInfo.IsDir() {
			chartArray[i].DeviceName = fileInfo.Name()
			csvFile := filepath.Join(csvDirectory, fileInfo.Name()+".csv")
			file, err := os.Open(csvFile)
			checkError(err, "Can't open csv file", true)
			reader := bufio.NewReader(file)

			for {
				line, fileErr := reader.ReadString('\n')
				timeStampArray := strings.Split(line, ",")
				dateTime, timeErr := time.ParseInLocation("2006-01-02 15:04:05", timeStampArray[0]+" "+timeStampArray[1], time.UTC)

				if timeErr != nil {
					dateTime = time.Now()
				}

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

				if fileErr != nil {
					break
				}
			}
		}
	}

	mu.Unlock()
	sendPayload(w, chartArray)
	return
}
