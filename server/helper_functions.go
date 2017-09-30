package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-ini/ini"
	"github.com/jmoiron/sqlx"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

// checkError is wrapper function to print custom error message along
// with stack trace along with ability to choose to exit program
func checkError(err error, message string, exit bool) {
	if err != nil {
		fmt.Printf("%+v\n", errors.Wrap(err, message))
		log.Printf("%+v\n", errors.Wrap(err, message))
		if exit {
			os.Exit(2)
		}
	}
}

// unableToRetrieveFiles is wrapper function for sending error message
// when we are trying to download tar files if error occurs
func unableToRetrieveFiles(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Could not retrieve files"))
	checkError(err, "", false)
}

// initFileSystem checks if project root dir exists and if it doesn't,
// we create proper dir/files for project and ask user for default values
// that will be written to server.ini config file
func initFileSystem() {
	_, err := os.Stat(projectRoot)

	if err != nil {
		os.MkdirAll(filepath.Join(setsDirectory), 0600)
		configFile, err := os.Create(serverConfigFile)
		checkError(err, "", true)
		_, err = os.Create(serverDBFile)
		checkError(err, "", true)
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("This is your first time running program.  We will now set some defaults.")
		fmt.Print("Enter ip address server will be listing on (default localhost):")
		ipAddress, _ := reader.ReadString('\n')
		fmt.Print("Enter port number you want the server to bind to (default :8003):")
		port, _ := reader.ReadString('\n')
		fmt.Print("Enter password you want to use for server (default password):")
		password, _ := reader.ReadString('\n')
		writeToFile :=
			"[DEFAULT] \n" +
				"# Ip address the server is given on local network \n" +
				"# Standard is 192.168.x.xx \n" +
				"ip_address=" + ipAddress + " \n\n" +

				"# Port that the server will listen on \n" +
				"port=" + port + " \n\n" +

				"# Password that will be used in post requests from device \n" +
				"# Should be the same as the client.ini file \n" +
				"password=" + password + " \n\n" +

				"# Determines if requests be made over https or not \n" +
				"# Strongly encouraged to have https as the password \n" +
				"# above will be sent in plain text though this requires \n" +
				"# setting up a ssl cert \n" +
				"# This setting must be the same in the server.ini \n" +
				"https=false \n\n" +

				"# Path to cert file \n" +
				"# If https is set to true, this has to be filled out \n" +
				"cert_file=" + projectRoot + "/ssl/cert_file.crt \n\n" +

				"# Path to key file \n" +
				"# If https is set to true, this has to be filled out \n" +
				"key_file=" + projectRoot + "/ssl/key_file.crt \n\n" +

				"# The number (in seconds) that determines how long a device \n" +
				"# can be inactive for before it is considered not working \n" +
				"# and be considered not checked in \n" +
				"# This settings should always be more than the 'sleep' setting \n" +
				"# in client.ini \n" +
				"time_out=5"

		configFile.WriteString(writeToFile)
	}
}

// initProjectFilePaths initiates file paths for our global variables
func initProjectFilePaths() {
	path, err := homedir.Dir()
	checkError(err, "Couldn't get project root", true)
	projectRoot = filepath.Join(path, projectName)
	serverConfigFile = filepath.Join(projectRoot, "server.ini")
	serverDBFile = filepath.Join(projectRoot, "server.db")
	csvDirectory = filepath.Join(projectRoot, "csv")
	setsDirectory = filepath.Join(csvDirectory, "sets")
}

// initLogger initiates logger and tells where to store logger file
func initLogger() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	f, err := os.OpenFile(
		filepath.Join(projectRoot, "rapsberry_pi_server.log"),
		os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	checkError(err, "Couldn't init logger", true)
	defer f.Close()
	log.SetOutput(f)
}

// loadSettingsFile loads settings from config file and assigns
// values to our settings struct
func loadSettingsFile() {
	cfg, err := ini.Load(serverConfigFile)
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

// commandLineArgs grabs command line arguments and sets up enviroment
// based on arguments passed
func commandLineArgs() {
	wipePtr := flag.Bool("wipe", false, "Wipes out all csv files and database, basically to start fresh")
	flag.Parse()

	if *wipePtr {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("You are about to wipe all csv files and entries in database.  Are you sure you want to continue? (y/n)")
		text, _ := reader.ReadString('\n')
		text = strings.TrimRight(text, "\r\n")
		if text == "y" || text == "Y" {
			// Extra saftey to make sure we don't delete home directory
			homeDir, _ := homedir.Dir()
			if projectRoot != homeDir {
				os.RemoveAll(projectRoot)
			} else {
				fmt.Println("Can't delete home directory, thats dangerous!")
			}
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

// execTXQuery is wrapper for executing an atomic transaction againt a database
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
		return err
	}

	tx.Commit()

	return nil
}

// initDatabase creates sqlite file and device table if they don't exist
func initDatabase() {
	_, err := os.Stat(serverDBFile)

	if err != nil {
		os.Create(serverDBFile)
	}

	db, err = sqlx.Open("sqlite3", serverDBFile)
	checkError(err, "Connecting to database", true)

	sqlQuery := "CREATE TABLE IF NOT EXISTS `device` (" +
		"`pk`					INTEGER PRIMARY KEY AUTOINCREMENT," +
		"`name`					TEXT UNIQUE," +
		"`set_num`				INTEGER," +
		"`lastest_set_time`		DATETIME NULL," +
		"`latest_check_in_time`	DATETIME," +
		"`is_new_set`			INTEGER," +
		"`is_recording`			INTEGER," +
		"`is_checked_in`		INTEGER" +
		");"

	_, err = db.Exec(sqlQuery)
	checkError(err, "Executing query", true)
}

// initGlobalVariables initiates global variables
func initGlobalVariables() {
	tpl = template.Must(template.ParseGlob("templates/*.html"))
	server = &http.Server{
		Addr:              setting.IPAddress + setting.Port,
		ReadTimeout:       (2 * time.Minute),
		ReadHeaderTimeout: (2 * time.Minute),
	}
	devices := make([]*device, 0)
	dbQuery := "SELECT * FROM device"
	err := db.Select(&devices, dbQuery, nil)
	checkError(err, "Improper query", true)
	counter := 0

	for _, item := range devices {
		counter++
		deviceCenter.Devices[item.Name] = item
	}

	deviceCenter.NumOfDevices = counter
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

func randomString(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
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
	duration := time.Duration(-setting.TimeOut) * time.Second

	for {
		now := time.Now().UTC()
		fmt.Println("update checkin")
		deviceCenter.Lock()

		for deviceName, dev := range deviceCenter.Devices {
			if dev.LatestCheckInTime.Before(now.Add(duration)) {
				fmt.Println("not heard from " + deviceName)
				query := "UPDATE device SET is_checked_in=0 WHERE name=?;"
				err := execTXQuery(query, deviceName)
				checkError(err, "", true)
				deviceCenter.Devices[deviceName].IsCheckedIn = false
			}
		}

		deviceCenter.Unlock()
		time.Sleep(sleepDuration)
	}
}
