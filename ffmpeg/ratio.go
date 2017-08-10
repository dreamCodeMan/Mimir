package ffmpeg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

const (
	INPUT_EMPTY_ERROR = "10006"
)

type Ratio struct {
	Ffmpeg FFMPEG `json:"video"`
	// Ftype 请求类型 0: 获取视频分辨率 1: 修改视频分辨率
	// Ftype int `json:"type"`
	// Fratio int    `json:"ratio"`
}

type RatioResponse struct {
	// Fratio 视频分辨率
	Fratio string `json:"ratio"`
}

// GetVideoRatio 获取指定视频的分辨率
func GetVideoRatio(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, GET_BODY_ERROR)
		return
	}

	var ratio Ratio

	err = json.Unmarshal(content, &ratio)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_BODY2JSON_ERROR)
		return
	}

	if ratio.Ffmpeg.Finput == "" {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, INPUT_EMPTY_ERROR)
		return
	}

	var ratioRespon RatioResponse

	ratioRespon.Fratio, err = _getVideoRatio(ratio.Ffmpeg.Finput)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, err.Error())
		return
	}

	respon, err := json.Marshal(ratioRespon)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_JSON2BODY_ERROR)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(respon))

	fmt.Println("Video Ratio Operation Finish")
	return
}
