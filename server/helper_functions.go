package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-ini/ini"
	"github.com/pkg/errors"
)

func checkError(err error, message string, exit bool) {
	if err != nil {
		fmt.Printf("%+v\n", errors.Wrap(err, message))

		if exit {
			os.Exit(2)
		}
	}
}

func initLogger() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func loadSettingsFile() {
	cfg, err := ini.Load("server.ini")
	checkError(err, "server.ini file not found", true)
	defaultSection, err := cfg.GetSection("DEFAULT")
	checkError(err, "No default section in ini file", true)
	ipAddress, err := defaultSection.GetKey("ip_address")
	checkError(err, "ip address setting not set", true)
	password, err := defaultSection.GetKey("password")
	checkError(err, "password setting is not set", true)
	https, err := defaultSection.GetKey("https")
	checkError(err, "https setting not set", true)
	boolHTTPS, err := strconv.ParseBool(https.Value())
	checkError(err, "https setting is not bool", true)
	timeOut, err := defaultSection.GetKey("time_out")
	checkError(err, "time_out setting not set", true)
	intTimeOut, err := strconv.ParseInt(timeOut.Value(), 10, 32)
	checkError(err, "timeout is not an int", true)
	port, err := defaultSection.GetKey("port")
	checkError(err, "port setting is not set", true)

	if boolHTTPS {
		certFile, err := defaultSection.GetKey("cert_file")
		checkError(err, "If https setting is true, cert_file must be set", true)
		keyFile, err := defaultSection.GetKey("key_file")
		checkError(err, "If https setting is true, key_file must be set", true)

		_, err = os.Stat(certFile.Value())
		checkError(err, "cert file does not exists", true)
		_, err = os.Stat(keyFile.Value())
		checkError(err, "key file does not exists", true)

	}

	setting = settings{
		IPAddress: ipAddress.Value(),
		Port:      port.Value(),
		Password:  password.Value(),
		HTTPS:     boolHTTPS,
		TimeOut:   intTimeOut,
	}
}

func commandLineArgs() {
	wipePtr := flag.Bool("wipe", false, "Wipes out all csv files and database, basically to start fresh")
	flag.Parse()

	if *wipePtr {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("You are about to wipe all csv files and entries in database.  Are you sure you want to continue? (y/n)")
		text, _ := reader.ReadString('\n')
		text = strings.TrimRight(text, "\r\n")
		if text == "y" || text == "Y" {
			os.RemoveAll("csv")
			os.Remove("server.db")
			fmt.Println("Server wiped")
		} else {
			fmt.Println("Files not deleted")
		}
	}
}

// getRoot gets the absolute current working directory
func getRoot() string {
	root, err := os.Getwd()
	checkError(err, "Can't get root", true)
	return root
}

func execTXQuery(query string, args ...interface{}) (err error) {
	stmt, err := db.Prepare(query)
	if err != nil {
		return err
	}

	tx, err := db.Begin()

	if err != nil {
		return err
	}

	_, err = tx.Stmt(stmt).Exec(args...)

	if err != nil {
		fmt.Println("doing rollback")
		log.Println(err)
		tx.Rollback()
	} else {
		tx.Commit()
	}

	return nil
}

func initDatabase() {
	var err error
	serverDB := "server.db"
	_, err = os.Stat(serverDB)

	if err != nil {
		os.Create(serverDB)
	}

	db, err = sql.Open("sqlite3", serverDB)
	checkError(err, "Connecting to database", true)

	sqlQuery := "CREATE TABLE IF NOT EXISTS `device_status` (" +
		"`pk`					INTEGER PRIMARY KEY AUTOINCREMENT," +
		"`device_name`			TEXT UNIQUE," +
		"`device_set`			INTEGER," +
		"`device_time`			TEXT," +
		"`is_new_set`			INTEGER," +
		"`is_recording`			INTEGER," +
		"`is_device_checked_in`	INTEGER" +
		");"

	_, err = db.Exec(sqlQuery)
	checkError(err, "Executing query", true)
}

func initGlobalVariables() {
	tpl = template.Must(template.ParseGlob("templates/*.html"))
	server = &http.Server{
		Addr:              setting.IPAddress + setting.Port,
		ReadTimeout:       (2 * time.Minute),
		ReadHeaderTimeout: (2 * time.Minute),
	}
	deviceCenter = &devices{
		DeviceSet:         make(map[string]int),
		DeviceTime:        make(map[string]time.Time),
		IsNewDeviceSet:    make(map[string]bool),
		IsDeviceRecording: make(map[string]bool),
		IsDeviceCheckedIn: make(map[string]bool),
	}

	err := os.MkdirAll(setsDirectoryPath, os.ModePerm)
	checkError(err, "Can't make directories", true)

	dbQuery :=
		"SELECT ds.device_name, ds.device_set, ds.device_time, ds.is_new_set, ds.is_recording, ds.is_device_checked_in " +
			"FROM device_status AS ds"
	rows, err := db.Query(dbQuery)
	checkError(err, "Start up query error", true)

	var deviceName string
	var deviceSet int
	var deviceTime string
	var isNewSet bool
	var isRecording bool
	var isDeviceCheckedIn bool

	for rows.Next() {
		err := rows.Scan(&deviceName, &deviceSet, &deviceTime, &isNewSet, &isRecording, &isDeviceCheckedIn)

		if err != nil {
			log.Fatal(err)
		}

		// arrayTime := strings.Split(deviceTime, ".")
		// convertedTime, err := time.Parse("2006-01-02 15:04:05", arrayTime[0])

		// if err != nil {
		// 	log.Fatal(err)
		// }

		deviceCenter.DeviceNames = append(deviceCenter.DeviceNames, deviceName)
		deviceCenter.DeviceSet[deviceName] = deviceSet
		deviceCenter.DeviceTime[deviceName] = time.Now()
		deviceCenter.IsDeviceRecording[deviceName] = isRecording
		deviceCenter.IsNewDeviceSet[deviceName] = isNewSet
		deviceCenter.IsDeviceCheckedIn[deviceName] = isDeviceCheckedIn
	}
}

// sendPayload is helper function that takes an empty interface
// and converts it to json and writes to to http.ResponseWriter
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

	if password != setting.Password {
		message = "Wrong Password"
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(message))
		return errors.New(message)
	}

	return nil
}

// updateCheckIn will be run on a seperate go routine and will loop
// through deviceCenter to see if any device have not been heard from
// based on the timeOut setting.  If a device hasn't been heard from
// based on timeOut, we change deviceCenter check in based on device
// The updateStatusHandler api end point is used in conjunction with
// this function as this function changes check in status for device and
// updateStatusHandler will use check in status to display message
// on webpage
func updateCheckIn() {
	sleepDuration := time.Duration(setting.TimeOut) * time.Second

	for {
		devicesNotHeardFrom := make(map[string]time.Time)
		fmt.Println("update checkin")
		deviceCenter.RLock()

		for deviceName, deviceTime := range deviceCenter.DeviceTime {
			duration := time.Duration(-setting.TimeOut) * time.Second
			// if deviceTime.Before(time.Now().Add(duration)) && deviceCenter.IsDeviceRecording[deviceName] {
			if deviceTime.Before(time.Now().Add(duration)) {
				fmt.Println("not heard from " + deviceName)
				devicesNotHeardFrom[deviceName] = deviceTime
			} else {
				deviceCenter.IsDeviceCheckedIn[deviceName] = true
			}
		}

		deviceCenter.RUnlock()
		deviceCenter.Lock()

		for deviceName := range devicesNotHeardFrom {
			deviceCenter.IsDeviceCheckedIn[deviceName] = false
		}

		deviceCenter.Unlock()
		fmt.Println(deviceCenter.IsDeviceCheckedIn)
		time.Sleep(sleepDuration)
	}
}
