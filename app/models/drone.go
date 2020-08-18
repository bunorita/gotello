package models

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"io"
	"io/ioutil"
	"log"
	"math"
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
	faceDetectXMLFile = "./app/models/haarcascade_frontalface_default.xml"
	snapshotDir       = "./static/img/snapshots/"
)

type DroneManager struct {
	*tello.Driver
	Speed                int
	patrolSem            *semaphore.Weighted
	patrolQuit           chan bool
	isPatrolling         bool
	ffmpegIn             io.WriteCloser
	ffmpegOut            io.ReadCloser
	Stream               *mjpeg.Stream
	faceDetectTrackingOn bool
	isSnapshot           bool
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
		classifier := gocv.NewCascadeClassifier()
		defer classifier.Close()
		if !classifier.Load(faceDetectXMLFile) {
			log.Printf("Failed to load cascade file: %s\n", faceDetectXMLFile)
			return
		}
		blue := color.RGBA{0, 0, 255, 0}

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

			if d.faceDetectTrackingOn {
				d.StopPatrol()
				rects := classifier.DetectMultiScale(img)
				log.Printf("found %d faces\n", len(rects))
				if len(rects) == 0 {
					d.Hover()
				}
				for _, r := range rects {
					gocv.Rectangle(&img, r, blue, 3)
					pt := image.Pt(r.Max.X, r.Min.Y-5)
					gocv.PutText(&img, "Human", pt, gocv.FontHersheyPlain, 1.2, blue, 2)

					// track face
					faceWidth := r.Max.X - r.Min.X
					faceHight := r.Max.Y - r.Min.Y
					faceCenterX := r.Min.X + faceWidth/2
					faceCenterY := r.Min.Y + faceHight/2
					faceArea := faceWidth * faceHight
					diffX := frameCenterX - faceCenterX
					diffY := frameCenterY - faceCenterY
					facePercent := math.Round(float64(faceArea) / float64(frameArea) * 100)

					move := false
					if diffX < -20 {
						d.Right(15)
						move = true
					}
					if diffX > 20 {
						d.Left(15)
						move = true
					}
					if diffY < -30 {
						d.Down(25)
						move = true
					}
					if diffY > 30 {
						d.Up(25)
						move = true
					}
					if facePercent > 7.0 {
						d.Backward(10)
						move = true
					}
					if facePercent < 0.9 {
						d.Forward(10)
						move = true
					}
					if !move {
						d.Hover()
					}
					break
				}
			}

			jpegBuf, err := gocv.IMEncode(gocv.JPEGFileExt, img)

			// save image as snapshot
			if d.isSnapshot {
				backupFileName := snapshotDir + time.Now().Format(time.RFC3339) + ".jpg"
				ioutil.WriteFile(backupFileName, jpegBuf, 0644)
				snapshotFileName := snapshotDir + "snapshot.jpg"
				ioutil.WriteFile(snapshotFileName, jpegBuf, 0644)
				d.isSnapshot = false
			}

			if err != nil {
				log.Println(err)
			}
			d.Stream.UpdateJPEG(jpegBuf)
		}
	}(d)
}

func (d *DroneManager) TakeSnapshot() {
	d.isSnapshot = true
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		if !d.isSnapshot || ctx.Err() != nil {
			break
		}
	}
	d.isSnapshot = false
}

func (d *DroneManager) EnableFaceDetectTracking() {
	d.faceDetectTrackingOn = true
}

func (d *DroneManager) DisableFaceDetectTracking() {
	d.faceDetectTrackingOn = false
	d.Hover()
}
