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
	"time"
)

var level = 4
var width = 550
var delay = 8 // delay in hours, so that I can see daylight on my wallpaper during the day!

type latestData struct {
	Date string `json:"date"`
	File string `json:"file"`
}

func main() {
	// Define work directory and files
	var home = os.Getenv("HOME")
	var work_dir = home + "/.himawari"
	var image_file = work_dir + "/latest.png"
	var dat_file = work_dir + "/data"

	// Read the last date from dat_file if it exists
	data, err := ioutil.ReadFile(dat_file)
	oldData := new(latestData)
	if err != nil {
		data = nil
	} else {
		err = json.Unmarshal(data, oldData)
		if err != nil {
			oldData = nil
		}
	}

	// Get the data about the last image
	initData := new(latestData)
	err = getJson("http://himawari8-dl.nict.go.jp/himawari8/img/D531106/latest.json", initData)
	if err != nil {
		log.Fatal("error getting json :", err)
	}

	location, _ := time.LoadLocation("Asia/Tokyo")
	t, _ := time.ParseInLocation("2006-01-02 15:04:05", initData.Date, location)
	if oldData != nil && oldData.Date == initData.Date {
		log.Print("No new image, keeping the old one")
		os.Exit(0)
	}
	t = t.Add(time.Duration(-delay) * time.Hour)

	// Get all the chunks from the latest image and assemble them
	output := image.NewRGBA(image.Rect(0, 0, level*width, level*width))
	for x := 0; x < level; x++ {
		for y := 0; y < level; y++ {
			url := fmt.Sprintf("http://himawari8.nict.go.jp/img/D531106/%dd/%d/%s_%d_%d.png",
				level, width, t.Format("2006/01/02/150405"), x, y)

			var buff image.Image
			err = getPng(url, &buff)
			if err != nil {
				err = getPng(url, &buff)
				if err != nil {
					log.Fatal("Error getting png :", url)
				}
			}

			draw.Draw(output, image.Rect(x*width, y*width, (x+1)*width, (y+1)*width), buff, image.Point{0, 0}, draw.Src)
		}
	}

	// Write output to file
	out, err := os.Create(image_file)
	if err != nil {
		err := os.Mkdir(work_dir, 0755)
		if err != nil {
			log.Fatal("error creating :", err)
		}
		out, err = os.Create(image_file)
		if err != nil {
			log.Fatal("error creating :", err)
		}
	}
	defer out.Close()
	err = png.Encode(out, output)
	if err != nil {
		log.Fatal("Error writing output:", err)
	}

	// Write dat file
	buff, err := json.Marshal(initData)
	ioutil.WriteFile(dat_file, buff, 0644)

	// Exec feh
	cmd := exec.Command("feh", "--bg-max", image_file)
	err = cmd.Run()
	if err != nil {
		log.Fatal("Error launching feh :", err)
	}
}

func getJson(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func getPng(url string, image *image.Image) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	*image, err = png.Decode(r.Body)
	return err
}
