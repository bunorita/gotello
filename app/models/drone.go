package models

import (
	"log"
	"time"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/dji/tello"
	"golang.org/x/sync/semaphore"
)

const (
	DefaultSpeed      = 10
	WaitDroneStartSec = 5
)

type DroneManager struct {
	*tello.Driver
	Speed        int
	patrolSem    *semaphore.Weighted
	patrolQuit   chan bool
	isPatrolling bool
}

func NewDroneManager() *DroneManager {
	drone := tello.NewDriver("8889")
	manager := &DroneManager{
		Driver:       drone,
		Speed:        DefaultSpeed,
		patrolSem:    semaphore.NewWeighted(1),
		patrolQuit:   make(chan bool),
		isPatrolling: false,
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

func (d *DroneManager) Patrol() {
	go func() {
		isAcquire := d.patrolSem.TryAcquire(1)
		if !isAcquire {
			d.patrolQuit <- true
			d.isPatrolling = false
			return
		}
		defer d.patrolSem.Release(1)
		d.isPatrolling = true
		status := 0
		t := time.NewTicker(3 * time.Second)
		for {
			select {
			case <-t.C:
				d.Hover()
				switch status {
				case 1:
					log.Println("1 Forward")
					d.Forward(d.Speed)
				case 2:
					log.Println("2 Right")
					d.Right(d.Speed)
				case 3:
					log.Println("3 Backward")
					d.Backward(d.Speed)
				case 4:
					log.Println("4 Left")
					d.Left(d.Speed)
				case 5:
					log.Println("5 rest status")
					status = 0
				}
				status++
			case <-d.patrolQuit:
				t.Stop()
				d.Hover()
				d.isPatrolling = false
				return
			}
		}
	}()
}

func (d *DroneManager) StartPatrol() {
	if !d.isPatrolling {
		d.Patrol()
	}
}

func (d *DroneManager) StopPatrol() {
	if d.isPatrolling {
		d.Patrol()
	}
}
