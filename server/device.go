package main

import (
	"sync"
	"time"
)

type devices struct {
	sync.RWMutex
	NumOfDevices      int
	DeviceNames       []string
	IsDeviceCheckedIn map[string]bool
	IsDeviceRecording map[string]bool
	DeviceTime        map[string]time.Time
	LatestSet         map[string]time.Time
	DeviceSet         map[string]int
	IsNewDeviceSet    map[string]bool
}

// type device struct {
// 	sync.RWMutex
// 	NumOfDevices      int
// 	DeviceNames       []string
// 	IsDeviceCheckedIn map[string]bool
// 	IsDeviceRecording map[string]bool
// 	DeviceTime        map[string]time.Time
// 	LatestSet         map[string]time.Time
// 	DeviceSet         map[string]int
// 	IsNewDeviceSet    map[string]bool
// }

// type deviceCenter struct {
// 	NumOfDevices int
// 	Devices      []device
// }

type chart struct {
	DeviceName  string      `json:"deviceName"`
	TimeMeasure string      `json:"timeMeasure"`
	Axises      map[int]int `json:"axises"`
}

type chartRow struct {
	DeviceName string `json:"deviceName"`
	NumOfSets  int    `json:"numOfSets"`
	FileNames  string `json:"fileNames"`
	LatestSet  string `json:"latestSet"`
}

type settings struct {
	IPAddress string
	Port      string
	Password  string
	HTTPS     bool
	TimeOut   int64
}
