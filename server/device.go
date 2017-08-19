package main

import (
	"sync"
	"time"
)

type devices struct {
	sync.RWMutex
	DeviceNames       []string
	IsDeviceCheckedIn map[string]bool
	IsDeviceRecording map[string]bool
	DeviceTime        map[string]time.Time
	DeviceSet         map[string]int
	IsNewDeviceSet    map[string]bool
}

type chart struct {
	DeviceName  string      `json:"deviceName"`
	TimeMeasure string      `json:"timeMeasure"`
	Axises      map[int]int `json:"axises"`
}

type settings struct {
	IPAddress string
	Port      string
	Password  string
	HTTPS     bool
	TimeOut   int64
}
