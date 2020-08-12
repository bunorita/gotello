package models

import (
	"time"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/dji/tello"
)

const (
	DefaultSpeed      = 10
	WaitDroneStartSec = 5
)

type DroneManager struct {
	*tello.Driver
	Speed int
}

func NewDroneManager() *DroneManager {
	drone := tello.NewDriver("8889")
	manager := &DroneManager{
		Driver: drone,
		Speed:  DefaultSpeed,
	}
	work := func() {
		//TODO
	}
	robot := gobot.NewRobot("tello",
		[]gobot.Connection{},
		[]gobot.Device{drone},
		work,
	)
	go robot.Start()
	time.Sleep(WaitDroneStartSec * time.Second)
	return manager
}
