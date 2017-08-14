package main

import (
	"sync"
	"time"
)

type devices struct {
	sync.RWMutex
	DeviceRecording map[string]bool
	DeviceTime      map[string]time.Time
	DeviceSet       map[string]string
	NewDeviceSet    map[string]bool
}

type chart struct {
	DeviceName  string      `json:"deviceName"`
	TimeMeasure string      `json:"timeMeasure"`
	Axises      map[int]int `json:"axises"`
}
