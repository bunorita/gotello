package utils

import (
	"io"
	"log"
	"os"
)

func LoggingSettings(fileName string) {
	logfile, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("file=%s err=%s\n", fileName, err.Error())
	}
	w := io.MultiWriter(os.Stdout, logfile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(w)
}
