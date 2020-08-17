package models

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"

	"github.com/hybridgroup/mjpeg"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/dji/tello"
	"gocv.io/x/gocv"
	"golang.org/x/sync/semaphore"
)

const (
	DefaultSpeed      = 10
	WaitDroneStartSec = 5
	frameX            = 960 / 3
	frameY            = 720 / 3
	frameCenterX      = frameX / 2
	frameCenterY      = frameY / 2
	frameArea         = frameX * frameY
	frameSize         = frameArea * 3
)

type DroneManager struct {
	*tello.Driver
	Speed        int
	patrolSem    *semaphore.Weighted
	patrolQuit   chan bool
	isPatrolling bool
	ffmpegIn     io.WriteCloser
	ffmpegOut    io.ReadCloser
	Stream       *mjpeg.Stream
}

func NewDroneManager() *DroneManager {
	drone := tello.NewDriver("8889")

	ffmpeg := exec.Command("ffmpeg",
		"-hwaccel", "auto",
		"-hwaccel_device", "opencl",
		"-i", "pipe:0",
		"-pix_fmt", "bgr24",
		"-s", fmt.Sprintf("%dx%d", frameX, frameY),
		"-f", "rawvideo", "pipe:1")
	ffmpegIn, _ := ffmpeg.StdinPipe()
	ffmpegOut, _ := ffmpeg.StdoutPipe()

	manager := &DroneManager{
		Driver:       drone,
		Speed:        DefaultSpeed,
		patrolSem:    semaphore.NewWeighted(1),
		patrolQuit:   make(chan bool),
		isPatrolling: false,
		ffmpegIn:     ffmpegIn,
		ffmpegOut:    ffmpegOut,
		Stream:       mjpeg.NewStream(),
	}
	work := func() {
		if err := ffmpeg.Start(); err != nil {
			log.Println(err)
			return
		}

		drone.On(tello.ConnectedEvent, func(data interface{}) {
			log.Println("Connected")
			drone.StartVideo()
			drone.SetVideoEncoderRate(tello.VideoBitRateAuto)
			drone.SetExposure(0)

			gobot.Every(100*time.Millisecond, func() {
				drone.StartVideo()
			})
			manager.StreamVideo()
		})

		drone.On(tello.VideoFrameEvent, func(data interface{}) {
			pkt := data.([]byte)
			if _, err := ffmpegIn.Write(pkt); err != nil {
				log.Println(err)
			}
		})
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

func (d *DroneManager) StreamVideo() {
	go func(d *DroneManager) {
		for {
			buf := make([]byte, frameSize)
			if _, err := io.ReadFull(d.ffmpegOut, buf); err != nil {
				log.Println(err)
			}
			img, err := gocv.NewMatFromBytes(frameY, frameX, gocv.MatTypeCV8UC3, buf)
			if err != nil {
				log.Println(err)
			}
			if img.Empty() {
				continue
			}
			jpegBuf, err := gocv.IMEncode(gocv.JPEGFileExt, img)
			if err != nil {
				log.Println(err)
			}
			d.Stream.UpdateJPEG(jpegBuf)
		}
	}(d)
}
