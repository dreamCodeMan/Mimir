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

// const (
//
// )

type Logo struct {
	Ffmpeg FFMPEG `json:"video"`
	// Flog logo全路径
	Flogo string `json:"logo"`
	// Fpostion Logo坐标位置, 默认0:0
	Fpostion string `json:"postion"`
}

type LogoResponse struct {
	// Ftoken 视频标识码
	Ftoken string `json:"token"`
	// Fprogress 当前进度
	Fprogress float32 `json:"progress"`
}

// VideoAddLogo 在指定视频中添加Logo
func VideoAddLogo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, GET_BODY_ERROR)
		return
	}

	var logo Logo

	err = json.Unmarshal(content, &logo)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_BODY2JSON_ERROR)
		return
	}

	_tokenChan := make(chan string)
	go func() {
		outDir := filepath.Dir(logo.Ffmpeg.Finput)
		outDir += "/"

		if logo.Ffmpeg.Fouput == "" {
			file := filepath.Base(logo.Ffmpeg.Finput)
			name := strings.Split(file, ".")
			if len(name) <= 1 {
				logo.Ffmpeg.Fouput = outDir + file + "_logo"
			} else {
				n := strings.Join(name[:len(name)-1], ".")
				logo.Ffmpeg.Fouput = outDir + n + "_logo" + "." + name[len(name)-1]
			}
			fmt.Println(logo.Ffmpeg.Fouput)
		}

		if logo.Fpostion == "" {
			logo.Fpostion = "00:00"
		}

		token := getToken(FPS_TOKEN_SIZE)
		_tokenChan <- token

		_originTime, err := _getVideoLength(logo.Ffmpeg.Finput)
		if err != nil {
			_originTime = "00:00:01"
		}

		_logoToken := getLogoToken()
		_logoToken[token] = 0

		_originLong := _paserDurl(_originTime)

		// ffmpeg -y -i piantou.mp4 -vf "movie=logo.png [logo];[in][logo] overlay=10:10 [out]" piantou_logo.mp4
		command := "-i " + logo.Ffmpeg.Finput + " "
		command += ("-y -vf movie=" + logo.Flogo + "<SPACE>[logo];[in][logo]<SPACE>overlay=" + logo.Fpostion + "<SPACE>[out] ")
		command += logo.Ffmpeg.Fouput

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
						_dualTime := _paserDurl(_time[1])
						_logoToken[token] = (float32(_dualTime) / float32(_originLong))
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

	var logoRespon LogoResponse

	logoRespon.Ftoken = <-_tokenChan

	respon, err := json.Marshal(logoRespon)
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

// VideoLogoGet 获取Logo添加进度
func VideoLogoGet(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	token := p.ByName("token")
	_logoToken := getLogoToken()

	logo := &LogoResponse{
		Ftoken: token,
	}
	if _, ok := _logoToken[token]; ok {
		logo.Fprogress = _logoToken[token]
	} else {
		logo.Fprogress = -1
	}

	respon, err := json.Marshal(logo)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_JSON2BODY_ERROR)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(respon))

	return
}
