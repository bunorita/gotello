package main

import (
	"log"

	"github.com/bunorita/gotello/app/controllers"
	"github.com/bunorita/gotello/config"
	"github.com/bunorita/gotrading/utils"
)

func main() {
	utils.LoggingSettings(config.Config.LogFile)
	// droneManager := models.NewDroneManager()
	// droneManager.TakeOff()
	// time.Sleep(10 * time.Second)
	// droneManager.Patrol()
	// time.Sleep(30 * time.Second)
	// droneManager.Patrol()
	// time.Sleep(10 * time.Second)
	// droneManager.Land()

	log.Println(controllers.StartWebServer())
}
