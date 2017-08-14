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

type Black struct {
	Ffmpeg FFMPEG `json:"video"`
}

type BlackResponse struct {
	Ftoken    string  `json:"token"`
	Fprogress float32 `json:"progress"`
	Foutput   string  `json:"output"`
}

// AddBlackInVideo 在视频中增加黑边
func AddBlackInVideo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, GET_BODY_ERROR)
		return
	}

	var black Black

	err = json.Unmarshal(content, &black)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_BODY2JSON_ERROR)
		return
	}

	var cmds []string

	if black.Ffmpeg.Fouput == "" {
		outDir := filepath.Dir(black.Ffmpeg.Finput)
		outDir += "/"
		file := filepath.Base(black.Ffmpeg.Finput)
		name := strings.Split(file, ".")
		if len(name) <= 1 {
			black.Ffmpeg.Fouput = outDir + file + "_out.mp4"
		} else {
			n := strings.Join(name[:len(name)-1], ".")
			black.Ffmpeg.Fouput = outDir + n + "_out." + name[len(name)-1]
		}
	}

	_tokenChan := make(chan string)
	go func() {

		token := getToken(FPS_TOKEN_SIZE)
		_tokenChan <- token

		_originTime, err := _getVideoLength(black.Ffmpeg.Finput)
		if err != nil {
			_originTime = "00:00:01"
		}

		_blackToken := getBlackToken()
		_blackToken[token] = 0

		_originLong := _paserDurl(_originTime)

		command := "-i " + black.Ffmpeg.Finput
		command += " -vf scale=1280:534,pad=1280:720:0:93:black -y "
		command += black.Ffmpeg.Fouput

		cmds = append(cmds, command)

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
						_dualTime := _paserDurl(_time[1])
						_blackToken[token] = (float32(_dualTime) / float32(_originLong))
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

	var bs BlackResponse
	bs.Ftoken = <-_tokenChan
	bs.Foutput = black.Ffmpeg.Fouput

	respon, err := json.Marshal(bs)
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

// GetBlackProgress 获取黑边生成进度
func GetBlackProgress(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	token := p.ByName("token")
	_blackToken := getBlackToken()

	bs := &BlackResponse{
		Ftoken: token,
	}

	if _, ok := _blackToken[token]; ok {
		bs.Fprogress = _blackToken[token]
	} else {
		bs.Fprogress = -1
	}

	respon, err := json.Marshal(bs)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_JSON2BODY_ERROR)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(respon))

	return
}
