package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/andy-zhangtao/Mimir/ffmpeg"
	"github.com/julienschmidt/httprouter"
)

var _VERSION_ = "unknown"

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
	router.GET("/v1/", _testConnect)
	router.POST("/v1/video/shot", ffmpeg.VideoShot)
	router.POST("/v1/video/fps", ffmpeg.VideoFps)
	router.GET("/v1/video/fps/:token", ffmpeg.VideoFpsGet)
	router.POST("/v1/video/ratio", ffmpeg.GetVideoRatio)
	router.PUT("/v1/video/ratio", ffmpeg.MoidfyVideoRatio)
	router.GET("/v1/video/ratio/:token", ffmpeg.VideoRatioGet)
	router.POST("/v1/video/concat", ffmpeg.ConcatVideo)
	router.GET("/v1/video/concat/:token", ffmpeg.ConcatVideoGet)
	router.POST("/v1/video/logo", ffmpeg.VideoAddLogo)
	router.GET("/v1/video/logo/:token", ffmpeg.VideoLogoGet)
	router.POST("/v1/video/cut", ffmpeg.VideoCutOut)
	router.POST("/v1/video/concat/multi", ffmpeg.MultiConcatVideo)
	router.POST("/v1/video/audio", ffmpeg.GetAudioFromVideo)
	router.GET("/v1/video/audio/:token", ffmpeg.GetAudioProgress)
	router.PUT("/v1/video/audio", ffmpeg.MergeAudioWithVideo)
	router.POST("/v1/video/separate", ffmpeg.VideoSeparate)
	router.GET("/v1/video/black/:token", ffmpeg.GetBlackProgress)
	router.POST("/v1/video/black", ffmpeg.AddBlackInVideo)

	log.Fatal(http.ListenAndServe(":8080", router))
}

func _testConnect(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	fmt.Fprintf(w, "My Name Is LiLei! "+_VERSION_)
	return
}
