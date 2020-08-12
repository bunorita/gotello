package main

import (
	"log"

	"github.com/bunorita/gotello/config"
	"github.com/bunorita/gotrading/utils"
)

func main() {
	utils.LoggingSettings(config.Config.LogFile)
	log.Println("test")

	// droneManager := models.NewDroneManager()
	// droneManager.TakeOff()
	// time.Sleep(10 * time.Second)
	// droneManager.Land()
}
