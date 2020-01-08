package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"
)

const (
	level = 4
	width = 550
	delay = 8 // delay in hours, so that I can see daylight on my wallpaper during the day!
)

type latestData struct {
	Date string `json:"date"`
	File string `json:"file"`
}

func main() {
	// Define work directory and files
	var (
		home      = os.Getenv("HOME")
		workDir   = path.Join(home, ".himawari")
		imageFile = path.Join(workDir, "/latest.png")
		dataFile  = path.Join(workDir, "/data")
		data      []byte
		err       error
		oldData   = new(latestData)
		initData  = new(latestData)
		t         time.Time
		location  *time.Location
		output    *image.RGBA
		out       *os.File
	)

	// Read the last date from dataFile if it exists
	data, err = ioutil.ReadFile(dataFile)
	if err != nil {
		data = nil
	} else {
		err = json.Unmarshal(data, oldData)
		if err != nil {
			oldData = nil
		}
	}

	// Get the data about the last image
	err = getJSON("http://himawari8-dl.nict.go.jp/himawari8/img/D531106/latest.json", initData)
	if err != nil {
		log.Fatal("error getting json data from himawari server: ", err)
	}

	location, err = time.LoadLocation("Asia/Tokyo")
	if err != nil {
		log.Fatal("error loading Asia/Tokyo time zone data: ", err)
	}
	t, _ = time.ParseInLocation("2006-01-02 15:04:05", initData.Date, location)
	if oldData != nil && oldData.Date == initData.Date {
		log.Print("No new image, keeping the old one")
		os.Exit(0)
	}
	t = t.Add(time.Duration(-delay) * time.Hour)

	// Get all the chunks from the latest image and assemble them
	output = image.NewRGBA(image.Rect(0, 0, level*width, level*width))
	for x := 0; x < level; x++ {
		for y := 0; y < level; y++ {
			url := fmt.Sprintf("http://himawari8.nict.go.jp/img/D531106/%dd/%d/%s_%d_%d.png",
				level, width, t.Format("2006/01/02/150405"), x, y)

			var buff image.Image
			err = getPNG(url, &buff)
			if err != nil {
				err = getPNG(url, &buff)
				if err != nil {
					log.Fatal("Error getting png :", url)
				}
			}
			draw.Draw(output, image.Rect(x*width, y*width, (x+1)*width, (y+1)*width), buff, image.Point{0, 0}, draw.Src)
		}
	}

	// Write output to file
	out, err = os.Create(imageFile)
	if err != nil {
		err := os.Mkdir(workDir, 0755)
		if err != nil {
			log.Fatal("error creating output directory:", err)
		}
		out, err = os.Create(imageFile)
		if err != nil {
			log.Fatal("error creating output file:", err)
		}
	}
	defer out.Close()
	err = png.Encode(out, output)
	if err != nil {
		log.Fatal("Error writing output file:", err)
	}

	// Write dat file
	buff, err := json.Marshal(initData)
	ioutil.WriteFile(dataFile, buff, 0644)

	// Exec feh
	cmd := exec.Command("feh", "--bg-max", imageFile)
	err = cmd.Run()
	if err != nil {
		log.Fatal("Error launching feh :", err)
	}
}

func getJSON(url string, target interface{}) (err error) {
	var r *http.Response
	r, err = http.Get(url)
	if err == nil {
		defer r.Body.Close()
		err = json.NewDecoder(r.Body).Decode(target)
	}
	return
}

func getPNG(url string, image *image.Image) (err error) {
	var r *http.Response
	r, err = http.Get(url)
	if err == nil {
		defer r.Body.Close()
		*image, err = png.Decode(r.Body)
	}
	return
}
