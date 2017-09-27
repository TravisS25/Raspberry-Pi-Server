package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
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
	setsDirectory := filepath.Join("csv", "sets")
	fileInfoArray, err := ioutil.ReadDir(setsDirectory)

	if err != nil {
		log.Fatal()
	}

	chartRows := make([]chartRow, 0)

	for _, fileInfo := range fileInfoArray {
		deviceName := fileInfo.Name()
		deviceContents, err := ioutil.ReadDir(setsDirectory + "/" + deviceName)
		checkError(err, "Couldn't read dir", true)
		fileNames := make([]string, 0)

		for _, file := range deviceContents {
			fileName := strings.Split(file.Name(), ".")[0]
			fileNames = append(fileNames, fileName)
		}

		stringFileNames := strings.Join(fileNames, ",")
		row := chartRow{
			DeviceName: deviceName,
			NumOfSets:  len(deviceContents),
			FileNames:  stringFileNames,
			LatestSet:  fileInfo.ModTime().Format("2006-01-02 15:04:05"),
		}

		chartRows = append(chartRows, row)
	}

	context := map[string]interface{}{
		"deviceCenter": deviceCenter,
		"chartRows":    chartRows,
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

	deviceDirectory := filepath.Join("csv", "sets", deviceName)
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
				Mode: 0600,
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

	rootPath := filepath.Join("csv", "sets")
	rootDirArray, err := ioutil.ReadDir(rootPath)
	randomFileName := randomString(20)
	tempFilePath := filepath.Join("/tmp", randomFileName)
	mainFile, err := os.Create(tempFilePath + ".tar.gz")
	if err != nil {
		log.Fatalln(err)
	}
	defer mainFile.Close()
	// set up the gzip writer
	gw := gzip.NewWriter(mainFile)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, dirInfo := range rootDirArray {
		dirName := dirInfo.Name()
		dirPath := filepath.Join(rootPath, dirName)
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
				Mode: 0600,
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

	r.ParseForm()
	var sqlStatement string
	deviceName := r.Form.Get("deviceName")
	doesNameExist := false
	alreadyCheckedIn := false

	// Loop through current devices in deviceCenter to check if passed device
	// name is already checked in
	// If it is, we return with an error message
	// If the device name doesn't exist, we add it to our deviceCenter
	for _, centerDeviceName := range deviceCenter.DeviceNames {
		if centerDeviceName == deviceName {
			if deviceCenter.IsDeviceCheckedIn[centerDeviceName] {
				alreadyCheckedIn = true
			}

			doesNameExist = true
			break
		}
	}

	// If device is already checked in, return with warning message
	if alreadyCheckedIn {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("Already checked in"))
		return
	}

	// If device already exists, update database
	// Else insert the new device into database with default values
	if doesNameExist {
		sqlStatement = "UPDATE device_status SET device_time=?, is_device_checked_in=? WHERE device_name=?"

		deviceCenter.Lock()
		deviceCenter.DeviceTime[deviceName] = time.Now()
		deviceCenter.IsDeviceCheckedIn[deviceName] = true
		deviceCenter.Unlock()

		err := execTXQuery(
			sqlStatement,
			deviceCenter.DeviceTime[deviceName],
			deviceCenter.IsDeviceCheckedIn[deviceName],
			deviceName,
		)
		checkError(err, "Update query error", true)
	} else {
		sqlStatement =
			"INSERT INTO device_status (device_name, device_set, device_time, is_new_set, is_recording, is_device_checked_in) " +
				"VALUES (?,?,?,?,?,?);"

		deviceCenter.Lock()
		now := time.Now()
		deviceCenter.DeviceNames = append(deviceCenter.DeviceNames, deviceName)
		deviceCenter.DeviceSet[deviceName] = 1
		deviceCenter.DeviceTime[deviceName] = now
		deviceCenter.IsNewDeviceSet[deviceName] = false
		deviceCenter.IsDeviceRecording[deviceName] = true
		deviceCenter.IsDeviceCheckedIn[deviceName] = true
		deviceCenter.Unlock()

		err := execTXQuery(sqlStatement, deviceName, 1, now.Format("2006-01-02 15:04:05"), 0, 1, 1)

		if err != nil {
			log.Fatal(err)
		}
	}

	if !alreadyCheckedIn {
		deviceCenter.RLock()
		payLoad := map[string]interface{}{
			"hasNewSetNotRecording": deviceCenter.IsNewDeviceSet[deviceName],
			"isRecording":           deviceCenter.IsDeviceRecording[deviceName],
			"deviceSet":             deviceCenter.DeviceSet[deviceName],
		}
		deviceCenter.RUnlock()
		sendPayload(w, payLoad)
	}

	// If current request is from new device, create directory with device
	// name under the sets directory
	err = os.MkdirAll(filepath.Join("csv", "sets", deviceName), os.ModePerm)
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
	chartArray := make([]chartRow, 0)
	sqlUpdate := "UPDATE device_status SET is_new_set=1, device_set=?, latest_set=? WHERE device_status.device_name=?"

	for _, deviceName := range r.Form["new-set"] {
		deviceCenter.RLock()
		// Check if device is even registered
		isNewDeviceSet, ok := deviceCenter.IsNewDeviceSet[deviceName]
		isDeviceRecording, _ := deviceCenter.IsDeviceRecording[deviceName]
		deviceCenter.RUnlock()

		// If device is not registered, return and add error message
		// Else if the device is still considered in "new set" mode
		// return with error message as this indicates that the device has not
		// signaled back that it has started it's new set locally
		// Else
		if !ok {
			message += deviceName + " is not registered <br /> "
			continue
		} else {
			if isDeviceRecording {
				message += deviceName + " is recording.  Can only start new set when " +
					"device is NOT recording <br /> "
				continue
			}
			if isNewDeviceSet {
				message += deviceName + " still hasn't reset to new set <br /> "
				continue
			}

			mu.Lock()
			currentCSVFilePath := filepath.Join("csv", deviceName+".csv")
			deviceSetDirectory := filepath.Join(setsDirectoryPath, deviceName)
			fileInfoArray, err := ioutil.ReadDir(deviceSetDirectory)

			if err != nil {
				log.Fatal(err)
			}

			currentCSVFile, err := os.Open(currentCSVFilePath)

			if err != nil {
				log.Fatal(err)
			}

			// If device directory has no files in it, then we insert the first
			// csv file in the set
			// Else calculate what the next file name should be.  Since file names
			// are just numbers, we just simply increment from the last file name
			if len(fileInfoArray) == 0 {
				newFile, err = os.OpenFile(filepath.Join(deviceSetDirectory, "1.csv"), os.O_WRONLY|os.O_CREATE, os.ModePerm)

				if err != nil {
					log.Fatal(err)
				}

				// deviceSet[deviceName] = "1"
				chartArray = append(chartArray, chartRow{
					DeviceName: deviceName,
					NumOfSets:  1,
					LatestSet:  time.Now().Format("2006-01-02 15:04:05"),
				})
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

				// deviceSet[deviceName] = stringFileName
				chartArray = append(chartArray, chartRow{
					DeviceName: deviceName,
					NumOfSets:  newFileName,
					LatestSet:  time.Now().Format("2006-01-02 15:04:05"),
				})
				_, err = io.Copy(newFile, currentCSVFile)

				if err != nil {
					log.Fatal(err)
				}
			}

			newFile.Close()
			currentCSVFile.Close()

			// Simply calling create to overwrite current file
			_, err = os.Create(currentCSVFilePath)

			if err != nil {
				log.Fatal(err)
			}

			mu.Unlock()

			now := time.Now()
			deviceCenter.Lock()
			deviceCenter.IsNewDeviceSet[deviceName] = true
			deviceCenter.DeviceSet[deviceName]++
			deviceCenter.LatestSet[deviceName] = now
			deviceCenter.Unlock()
			execTXQuery(sqlUpdate, deviceCenter.DeviceSet[deviceName], now.Format("2006-01-02 15:04:05"), deviceName)
		}
	}

	sendPayload(w, map[string]interface{}{
		"chartArray": chartArray,
		"message":    message,
	})
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
	sqlUpdate := "UPDATE device_status SET is_recording=? WHERE device_status.device_name=?"
	if deviceNames, ok := r.Form["record-device"]; ok {
		for _, deviceName := range deviceNames {
			if _, ok := deviceCenter.IsDeviceRecording[deviceName]; ok {
				if record == "true" {
					deviceCenter.IsDeviceRecording[deviceName] = true
				} else {
					deviceCenter.IsDeviceRecording[deviceName] = false
				}
				execTXQuery(sqlUpdate, deviceCenter.IsDeviceRecording[deviceName], deviceName)
			}
		}
	}

	deviceCenter.Unlock()
	sendPayload(w, deviceCenter.IsDeviceRecording)
}

// updateStatusHandler is an api endpoint that checks the statuses of
// all devices and if a device hasn't been heard from for certain
// amount of time, will add device name to list and give warning on webpage
func updateStatusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("record status")
	devicesNotHeardFrom := make(map[string]time.Time)
	chartRows := make([]chartRow, 0)
	deviceCenter.RLock()

	for _, deviceName := range deviceCenter.DeviceNames {
		if !deviceCenter.IsDeviceCheckedIn[deviceName] {
			devicesNotHeardFrom[deviceName] = deviceCenter.DeviceTime[deviceName]
		}
		chartRows = append(chartRows, chartRow{
			DeviceName: deviceName,
			NumOfSets:  deviceCenter.DeviceSet[deviceName] - 1,
			LatestSet:  deviceCenter.LatestSet[deviceName].Format("2006-01-02 15:04:05"),
		})
	}

	deviceCenter.RUnlock()

	// for deviceName, deviceTime := range deviceCenter.DeviceTime {
	// 	duration := time.Duration(-setting.TimeOut) * time.Second
	// 	if deviceTime.Before(time.Now().Add(duration)) && deviceCenter.IsDeviceRecording[deviceName] {
	// 		devicesNotHeardFrom[deviceName] = deviceTime
	// 	}
	// }

	// deviceCenter.RUnlock()
	// deviceCenter.Lock()

	// for deviceName := range devicesNotHeardFrom {
	// 	deviceCenter.IsDeviceCheckedIn[deviceName] = false
	// }
	// fmt.Println(deviceCenter.IsDeviceCheckedIn)
	// deviceCenter.Unlock()
	sendPayload(w, map[string]interface{}{
		"chartRows":           chartRows,
		"devicesNotHeardFrom": devicesNotHeardFrom,
	})
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
	record, recordOK := deviceCenter.IsDeviceRecording[deviceName]
	newSet, newSetOK := deviceCenter.IsNewDeviceSet[deviceName]
	deviceCenter.RUnlock()

	if recordOK || newSetOK {
		w.WriteHeader(http.StatusOK)
		now := time.Now()
		deviceCenter.Lock()
		deviceCenter.DeviceTime[deviceName] = now
		deviceCenter.Unlock()

		timeUpdateQuery := "UPDATE device_status SET device_time=? WHERE device_status.device_name=?;"
		execTXQuery(timeUpdateQuery, now.Format("2006-01-02 15:04:05"), deviceName)

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
	isCheckedIn := false
	timeStamp := r.Form.Get("timeStamp")
	deviceName := strings.Split(timeStamp, ",")[0]
	duration := time.Duration(-setting.TimeOut) * time.Second
	deviceCenter.RLock()
	deviceTime, ok := deviceCenter.DeviceTime[deviceName]
	deviceCenter.RUnlock()

	if ok {
		isCheckedIn = deviceTime.After(time.Now().Add(duration))
	}

	// If deviceName exists in deviceCenter
	if ok && isCheckedIn {
		deviceCenter.RLock()
		isRecording := deviceCenter.IsDeviceRecording[deviceName]
		isNewSet := deviceCenter.IsNewDeviceSet[deviceName]
		deviceCenter.RUnlock()

		deviceCenter.Lock()
		deviceCenter.DeviceTime[deviceName] = time.Now()
		deviceCenter.Unlock()

		// If device is issued to stop recording, send message to device
		// to stop recording
		// Else begin/continue recording
		if !isRecording {
			message += "Stop Recording,"
		} else {
			message += "Record,"
		}

		if isNewSet {
			deviceCenter.Lock()
			deviceCenter.IsNewDeviceSet[deviceName] = false
			deviceCenter.Unlock()
		}

		// Write message to device
		w.Write([]byte(message))
		sqlUpdate :=
			"UPDATE device_status " +
				"SET device_time=?, is_recording=? is_new_set=0" +
				"WHERE device_status.device_name=?;"

		execTXQuery(
			sqlUpdate,
			deviceCenter.DeviceTime[deviceName].Format("2006-01-02 15:04:05"),
			deviceCenter.IsDeviceRecording[deviceName],
			deviceName,
		)

		deviceFilePath := filepath.Join("csv", deviceName+".csv")
		movement, err := strconv.ParseBool(strings.TrimRight(strings.Split(timeStamp, ",")[3], "\n"))
		checkError(err, "Can't parse bool", true)
		_, deviceErr := os.Stat(deviceFilePath)

		// Only write to file if movement is detected
		if movement {
			mu.Lock()
			defer mu.Unlock()

			// If log file for device does not exist, create and write time stamp to it
			// Else append time stamp to file
			if deviceErr != nil {
				deviceFile, _ = os.Create(deviceFilePath)
				deviceFile.WriteString(timeStamp)
			} else {
				deviceFile, err = os.OpenFile(deviceFilePath, os.O_APPEND|os.O_WRONLY, os.ModePerm)
				checkError(err, "Can't open file", true)
				deviceFile.WriteString(timeStamp)
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
	now := time.Now()

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

	fileInfoArray, err := ioutil.ReadDir("csv")
	checkError(err, "Can't read files in directory", true)
	chartArray := make([]*chart, len(fileInfoArray))
	mu.RLock()

	for i, fileInfo := range fileInfoArray {
		if !fileInfo.IsDir() {
			chartArray[i].DeviceName = fileInfo.Name()
			csvFile := filepath.Join("csv", fileInfo.Name()+".csv")
			file, err := os.Open(csvFile)
			checkError(err, "Can't open csv file", true)
			reader := bufio.NewReader(file)

			for {
				line, err := reader.ReadString('\n')
				timeStampArray := strings.Split(line, ",")
				// movement, err := strconv.ParseBool(timeStampArray[3])
				checkError(err, "Can't parse bool", true)
				dateTime, timeErr := time.Parse("2006-06-01 11:20:10", timeStampArray[1]+" "+timeStampArray[2])

				if timeErr != nil {
					dateTime = time.Now()
				}

				// if movement {
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
				// }

				if err != nil {
					break
				}
			}
		}
	}

	mu.Unlock()
	sendPayload(w, chartArray)
	return
}
