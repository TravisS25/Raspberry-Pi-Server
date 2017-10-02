package main

import (
	"fmt"
	"html/template"
	"net/http"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var (
	mu               sync.RWMutex
	tpl              *template.Template
	deviceCenter     *devCenter
	db               *sqlx.DB
	server           *http.Server
	setting          settings
	projectRoot      string
	serverDBFile     string
	serverConfigFile string
	// serverLog        string
	csvDirectory  string
	setsDirectory string
)

const (
	projectName = ".raspberry_pi_server"
)

func init() {
	initProjectFilePaths()
	initFileSystem()
	initLogger()
	loadSettingsFile()
	commandLineArgs()
	initDatabase()
	initGlobalVariables()
}

func main() {
	fmt.Println("Server running...")

	http.HandleFunc("/", mainView)
	http.HandleFunc("/new-set/", newSetHandler)
	http.HandleFunc("/reload-csv/", reloadCSVHandler)
	http.HandleFunc("/record-mode-handler/", recordModeHandler)
	http.HandleFunc("/device-status-handler/", deviceStatusHandler)
	http.HandleFunc("/update-status-handler/", updateStatusHandler)
	http.HandleFunc("/sensor-handler/", sensorHandler)
	http.HandleFunc("/update-chart-handler/", updateChartHandler)
	http.HandleFunc("/check-in-handler/", deviceCheckInHandler)
	http.HandleFunc("/download-tar/", downloadTarHandler)
	http.HandleFunc("/generate-device-tar/", generateDeviceTarHandler)
	http.HandleFunc("/generate-all-devices-tar/", generateAllDevicesTarHandler)

	fmt.Println("here")
	go updateCheckIn()

	if setting.HTTPS {
		// server.Addr = "https://" + setting.IPAddress + setting.Port
		// err := server.ListenAndServeTLS()
		// checkError(err, "Listen and server tls", true)
	} else {
		err := server.ListenAndServe()
		checkError(err, "Listen and server", true)
	}
}
