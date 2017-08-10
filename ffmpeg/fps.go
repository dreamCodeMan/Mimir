package ffmpeg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

const (
	GET_VIDEO_FPS_ERROR = "10005"
	FPS_TOKEN_SIZE      = 12
)

type Fps struct {
	Ffmpeg FFMPEG `json:"video"`
	// Ffps 准备修改的帧率
	Ffps int `json:"fps"`
}

type FpsResponse struct {
	// Ffps 视频原始帧率
	Ffps int `json:"fps"`
	// Ftoken 视频唯一标示
	Ftoken    string  `json:"token"`
	Fprogress float32 `json:"progress"`
}

// VideoFpsGet 获取指定token的变帧进度
func VideoFpsGet(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	token := p.ByName("token")
	_fpsToken := getFpsToken()

	fps := &FpsResponse{
		Ftoken: token,
	}
	if _, ok := _fpsToken[token]; ok {
		fps.Fprogress = _fpsToken[token]
	} else {
		fps.Fprogress = -1
	}

	respon, err := json.Marshal(fps)
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

// VideoFps 修改指定视频的帧率
func VideoFps(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, GET_BODY_ERROR)
		return
	}

	var fps Fps

	err = json.Unmarshal(content, &fps)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_BODY2JSON_ERROR)
		return
	}

	_tokenChan := make(chan string)
	go func() {

		outDir := filepath.Dir(fps.Ffmpeg.Finput)
		outDir += "/"

		if fps.Ffmpeg.Fouput == "" {
			file := filepath.Base(fps.Ffmpeg.Finput)
			name := strings.Split(file, ".")
			if len(name) <= 1 {
				fps.Ffmpeg.Fouput = outDir + file + "_" + strconv.Itoa(fps.Ffps)
			} else {
				n := strings.Join(name[:len(name)-1], ".")
				fps.Ffmpeg.Fouput = outDir + n + "_" + strconv.Itoa(fps.Ffps) + "." + name[len(name)-1]
			}
			fmt.Println(fps.Ffmpeg.Fouput)
		}

		token := getToken(FPS_TOKEN_SIZE)
		_tokenChan <- token

		_originTime, err := _getVideoLength(fps.Ffmpeg.Finput)
		if err != nil {
			_originTime = "00:00:01"
		}

		_fpsToken := getFpsToken()
		_fpsToken[token] = 0

		_originLong := _paserDurl(_originTime)

		command := "-i " + fps.Ffmpeg.Finput + " "
		command += ("-y -r " + strconv.Itoa(fps.Ffps) + " ")
		command += fps.Ffmpeg.Fouput

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
						_fpsToken[token] = (float32(_dualTime) / float32(_originLong))
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

	var fpsRespon FpsResponse

	fpsRespon.Ffps, err = _getVideoFps(fps)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Println(err.Error())
		fmt.Fprintf(w, GET_VIDEO_FPS_ERROR)
		return
	}

	fpsRespon.Ftoken = <-_tokenChan

	respon, err := json.Marshal(fpsRespon)
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

// _paserDurl 将hh:mm:ss格式的时间转换为秒
func _paserDurl(dual string) int {
	_total := 0
	_time := strings.Split(dual, ":")
	_second, err := strconv.ParseFloat(_time[2], 64)
	if err != nil {
		_second = 0
	}

	_min, err := strconv.Atoi(_time[1])
	if err != nil {
		_min = 0
	}

	_total = _min*60 + int(_second)

	_hour, err := strconv.Atoi(_time[0])
	if err != nil {
		_hour = 0
	}

	_total += _hour * 60 * 60
	return _total
}

// func _countPress(duar string, _length string) int {

// }

// _getVideoFps 获取视频原始帧率
func _getVideoFps(fps Fps) (int, error) {

	command := "-i " + fps.Ffmpeg.Finput

	_dataChan := make(chan chan string)
	_errorChan := make(chan error)
	_exit := make(chan int)
	go func() {
		err := _exec(command, _dataChan)
		if err != nil {
			_errorChan <- err
		}
		_exit <- 1
	}()

	_chan := make(chan string)
	_dataChan <- _chan
	for {
		isExit := false
		select {
		case e := <-_errorChan:
			return 0, e
		case d := <-_chan:
			// fmt.Println(d)
			if strings.Contains(d, "fps,") {
				_split := strings.Split(d, " fps,")
				_info := strings.Split(_split[0], ",")
				_fps := _info[len(_info)-1]
				return strconv.Atoi(strings.TrimSpace(_fps))
			}
		case <-_exit:
			isExit = true
		}

		if isExit {
			break
		}
	}

	return 0, nil
}
