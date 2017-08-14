package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	mu           sync.RWMutex
	tpl          *template.Template
	deviceCenter *devices
	db           *sql.DB
)

const (
	timeOut     = -5
	logFilePath = "csv/logfile.csv"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var err error
	tpl = template.Must(template.ParseGlob("templates/*.html"))
	db, err = sql.Open("sqlite3", "server.db")

	if err != nil {
		log.Fatal(err)
	}

	initDeviceCenter()
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

	err := http.ListenAndServe(":8003", nil) // set listen port
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	http.ListenAndServe("192.168.1.3:8003", nil)
}
