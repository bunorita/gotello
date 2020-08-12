package main

import (
	"time"

	"github.com/bunorita/gotello/app/models"
)

func main() {
	droneManager := models.NewDroneManager()
	droneManager.TakeOff()
	time.Sleep(10 * time.Second)
	droneManager.Land()
}
