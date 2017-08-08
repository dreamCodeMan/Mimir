package main

import (
	"log"
	"net/http"

	"github.com/andy-zhangtao/Mimir/ffmpeg"
	"github.com/julienschmidt/httprouter"
)

func main() {
	os, err := GetOS()
	if err != nil {
		// 理论上不会出现这种情况
	}

	err = CheckFFmpeg()
	if err != nil {
		DownFFmpeg(os)
	}

	router := httprouter.New()
	router.POST("/v1/video/shot", ffmpeg.VideoShot)
	log.Fatal(http.ListenAndServe(":8080", router))
}
