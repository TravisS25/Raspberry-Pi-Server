package main

import (
	"sync"
)

// type devices struct {
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

// type device struct {
// 	Pk                int       `json:"pk" db:"pk"`
// 	Name              string    `json:"name" db:"name"`
// 	SetNum            int       `json:"setNum" db:"set_num"`
// 	LatestSetTime     time.Time `json:"latestSetTime" db:"latest_set_time"`
// 	LatestCheckInTime time.Time `json:"latestCheckInTime" db:"latest_check_in_time"`
// 	IsNewSet          bool      `json:"isNewSet" db:"is_new_set"`
// 	IsRecording       bool      `json:"isRecording" db:"is_recording"`
// 	IsCheckedIn       bool      `json:"isCheckedIn" db:"is_checked_in"`
// }

type device struct {
	Pk                int     `json:"pk" db:"pk"`
	Name              string  `json:"name" db:"name"`
	SetNum            int     `json:"setNum" db:"set_num"`
	LatestSetTime     *string `json:"latestSetTime" db:"latest_set_time"`
	LatestCheckInTime string  `json:"latestCheckInTime" db:"latest_check_in_time"`
	IsNewSet          bool    `json:"isNewSet" db:"is_new_set"`
	IsRecording       bool    `json:"isRecording" db:"is_recording"`
	IsCheckedIn       bool    `json:"isCheckedIn" db:"is_checked_in"`
}

type devCenter struct {
	sync.RWMutex
	NumOfDevices int
	Devices      map[string]*device
}

type chart struct {
	DeviceName  string      `json:"deviceName"`
	TimeMeasure string      `json:"timeMeasure"`
	Axises      map[int]int `json:"axises"`
}

// type chartRow struct {
// 	DeviceName string `json:"deviceName"`
// 	NumOfSets  int    `json:"numOfSets"`
// 	FileNames  string `json:"fileNames"`
// 	LatestSet  string `json:"latestSet"`
// }

type settings struct {
	IPAddress string
	Port      string
	Password  string
	HTTPS     bool
	TimeOut   int64
}
