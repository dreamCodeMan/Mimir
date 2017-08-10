package ffmpeg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/julienschmidt/httprouter"
)

const (
	INPUT_EMPTY_ERROR = "10006"
)

type Ratio struct {
	Ffmpeg FFMPEG `json:"video"`
	// Fratio 准备调整的分辨率
	Fratio string `json:"ratio"`
}

type RatioResponse struct {
	// Fratio 视频分辨率
	Fratio string `json:"ratio"`
	// Ftoken 视频标识码
	Ftoken string `json:"token"`
	// Fprogress 当前进度
	Fprogress float32 `json:"progress"`
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

// MoidfyVideoRatio 修改分辨率
func MoidfyVideoRatio(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
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

	_tokenChan := make(chan string)

	go func() {

		outDir := filepath.Dir(ratio.Ffmpeg.Finput)
		outDir += "/"

		if ratio.Ffmpeg.Fouput == "" {
			file := filepath.Base(ratio.Ffmpeg.Finput)
			name := strings.Split(file, ".")
			if len(name) <= 1 {
				ratio.Ffmpeg.Fouput = outDir + file + "_" + ratio.Fratio
			} else {
				n := strings.Join(name[:len(name)-1], ".")
				ratio.Ffmpeg.Fouput = outDir + n + "_" + ratio.Fratio + "." + name[len(name)-1]
			}
			fmt.Println(ratio.Ffmpeg.Fouput)
		}

		token := getToken(FPS_TOKEN_SIZE)
		_tokenChan <- token

		_originTime, err := _getVideoLength(ratio.Ffmpeg.Finput)
		if err != nil {
			_originTime = "00:00:01"
		}

		_ratioToken := getRatioToken()
		_ratioToken[token] = 0

		_originLong := _paserDurl(_originTime)

		command := "-i " + ratio.Ffmpeg.Finput + " "
		command += ("-y -s " + ratio.Fratio + " ")
		command += ratio.Ffmpeg.Fouput

		cmds := []string{command}

		_dataChan := make(chan chan string)
		_errorChan := make(chan error)
		_exit := make(chan int)

		defer func() {
			close(_dataChan)
			close(_errorChan)
			close(_exit)
			if err := recover(); err != nil {
				fmt.Println(err)
			}
		}()

		go func() {
			for _, c := range cmds {
				err = _exec(c, _dataChan)
				if err != nil {
					_errorChan <- err
				}
				_exit <- 1
				fmt.Println("Exec控制协程退出")
			}

			return
		}()

		for _ = range cmds {
			_chan := make(chan string)
			_dataChan <- _chan

			isExit := false

			for {
				select {
				case s := <-_chan:
					if strings.Contains(s, " bitrate=") {
						_info := strings.Split(s, " bitrate=")
						_time := strings.Split(_info[0], "time=")
						// fmt.Println(_time[1])
						_dualTime := _paserDurl(_time[1])
						// fmt.Println(_time[1], _dualTime, _originLong)
						_ratioToken[token] = (float32(_dualTime) / float32(_originLong))
						// fmt.Println(_fpsToken[token])
					}
				case <-_exit:
					isExit = true
				}

				if isExit {
					break
				}
			}
		}

	}()

	var ratioRespon RatioResponse

	ratioRespon.Ftoken = <-_tokenChan

	respon, err := json.Marshal(ratioRespon)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_JSON2BODY_ERROR)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(respon))

	fmt.Println("Video Operation Finish")
	return
}

// VideoRatioGet 获取视频修改分辨率进度
func VideoRatioGet(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	token := p.ByName("token")
	_ratioToken := getRatioToken()

	ratio := &RatioResponse{
		Ftoken: token,
	}
	if _, ok := _ratioToken[token]; ok {
		ratio.Fprogress = _ratioToken[token]
	} else {
		ratio.Fprogress = -1
	}

	respon, err := json.Marshal(ratio)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_JSON2BODY_ERROR)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(respon))

	// fmt.Println("Video Operation Finish")
	return
}
