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
	AUDIO_EMPTY_ERROR = "10010"
)

// Audio 保存音轨信息
type Audio struct {
	Ffmpeg FFMPEG `json:"video"`
	// Faudio 当音轨和视频进行合并时，此属性为必填
	Faudio string `json:"audio"`
}

type AudioResponse struct {
	Output string `json:"audio"`
	// 视频唯一标示
	Ftoken string `json:"token"`
	// Fprogress 音频合并进度
	Fprogress float32 `json:"progress"`
}

// GetAudioProgress 获取音频合并进度
func GetAudioProgress(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	token := p.ByName("token")
	_audioToken := getAudioToken()

	as := &AudioResponse{
		Ftoken: token,
	}

	if _, ok := _audioToken[token]; ok {
		as.Fprogress = _audioToken[token]
	} else {
		as.Fprogress = -1
	}

	respon, err := json.Marshal(as)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_JSON2BODY_ERROR)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(respon))

	return
}

// VideoSeparate 分离指定视频流
func VideoSeparate(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, GET_BODY_ERROR)
		return
	}

	var audio Audio

	err = json.Unmarshal(content, &audio)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_BODY2JSON_ERROR)
		return
	}

	var cmds []string

	if audio.Ffmpeg.Fouput == "" {
		outDir := filepath.Dir(audio.Ffmpeg.Finput)
		outDir += "/"
		file := filepath.Base(audio.Ffmpeg.Finput)
		name := strings.Split(file, ".")
		if len(name) <= 1 {
			audio.Ffmpeg.Fouput = outDir + file + "_out.mp4"
		} else {
			n := strings.Join(name[:len(name)-1], ".")
			audio.Ffmpeg.Fouput = outDir + n + "_out." + name[len(name)-1]
		}
	}

	// _originTime, err := _getVideoLength(audio.Ffmpeg.Finput)
	// if err != nil {
	// 	_originTime = "00:00:01"
	// }

	command := " -i " + audio.Ffmpeg.Finput
	command += " -y -vcodec copy -an "
	command += audio.Ffmpeg.Fouput

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

	var as AudioResponse

	for _ = range cmds {
		_chan := make(chan string)
		_dataChan <- _chan

		isExit := false

		for {
			select {
			case <-_chan:
			case <-_exit:
				isExit = true
			}

			if isExit {
				break
			}
		}
	}

	as.Output = audio.Ffmpeg.Fouput

	respon, err := json.Marshal(as)
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

// MergeAudioWithVideo 将音轨与视频进行合并
func MergeAudioWithVideo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, GET_BODY_ERROR)
		return
	}

	var audio Audio
	var cmds []string

	err = json.Unmarshal(content, &audio)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_BODY2JSON_ERROR)
		return
	}

	if audio.Faudio == "" {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, AUDIO_EMPTY_ERROR)
		return
	}

	if audio.Ffmpeg.Fouput == "" {
		audio.Ffmpeg.Fouput = audio.Ffmpeg.Finput
	}

	_tokenChan := make(chan string)
	go func() {

		token := getToken(FPS_TOKEN_SIZE)
		_tokenChan <- token

		_originTime, err := _getVideoLength(audio.Ffmpeg.Finput)
		if err != nil {
			_originTime = "00:00:01"
		}

		_audioToken := getAudioToken()
		_audioToken[token] = 0

		_originLong := _paserDurl(_originTime)

		command := "-i " + audio.Ffmpeg.Finput
		command += " -i " + audio.Faudio + " -y "
		command += audio.Ffmpeg.Fouput

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
						_audioToken[token] = (float32(_dualTime) / float32(_originLong))
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

	var as AudioResponse
	as.Ftoken = <-_tokenChan

	respon, err := json.Marshal(as)
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

// GetAudioFromVideo 从视频中分离出音轨
func GetAudioFromVideo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, GET_BODY_ERROR)
		return
	}

	var audio Audio

	err = json.Unmarshal(content, &audio)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_BODY2JSON_ERROR)
		return
	}

	var cmds []string

	if audio.Ffmpeg.Fouput == "" {
		outDir := filepath.Dir(audio.Ffmpeg.Finput)
		outDir += "/"
		file := filepath.Base(audio.Ffmpeg.Finput)
		name := strings.Split(file, ".")
		if len(name) <= 1 {
			audio.Ffmpeg.Fouput = outDir + file + ".mp3"
		} else {
			n := strings.Join(name[:len(name)-1], ".")
			audio.Ffmpeg.Fouput = outDir + n + ".mp3"
		}
	}

	command := "-i " + audio.Ffmpeg.Finput
	command += " -y -vn -ar 44100 -ac 2 -ab 192 -f mp3 "
	command += audio.Ffmpeg.Fouput

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

	var as AudioResponse

	for _ = range cmds {
		_chan := make(chan string)
		_dataChan <- _chan

		isExit := false

		for {
			select {
			case <-_chan:
			case <-_exit:
				isExit = true
			}

			if isExit {
				break
			}
		}
	}

	as.Output = audio.Ffmpeg.Fouput

	respon, err := json.Marshal(as)
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
