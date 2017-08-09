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

	// go func() {
	// 	for {
	// 		fmt.Println(runtime.NumGoroutine())
	// 		time.Sleep(time.Duration(5) * time.Second)
	// 	}

	// }()

	router := httprouter.New()
	router.POST("/v1/video/shot", ffmpeg.VideoShot)
	router.POST("/v1/video/fps", ffmpeg.VideoFps)

	log.Fatal(http.ListenAndServe(":8080", router))
}
