package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	mu           sync.RWMutex
	tpl          *template.Template
	deviceCenter *devices
	db           *sql.DB
	server       *http.Server
	setting      settings
)

const (
	setsDirectoryPath = "csv/sets"
)

func init() {
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
