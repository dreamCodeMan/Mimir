package ffmpeg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

const (
	MAX_VIDEO_LEN             = 5
	OUTPUT_EMPTY_ERROR        = "10007"
	TOO_MANY_CONCAT_REQ_ERROR = "10009"
)

// Concat 视频合并
type Concat struct {
	Finput  []string `json:"input"`
	Foutput string   `json:"output"`
	// 是否需要随机拼接,默认为False
	FisRandom bool `json:"random"`
}

type ConcatResponse struct {
	// Ftoken 视频标识码
	Ftoken string `json:"token"`
	// Fprogress 当前进度
	Fprogress float32 `json:"progress"`
}

// ConcatVideoGet 获取视频合并进度
func ConcatVideoGet(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	token := p.ByName("token")
	_concatToken := getConcatToken()

	concat := &ConcatResponse{
		Ftoken: token,
	}
	if _, ok := _concatToken[token]; ok {
		concat.Fprogress = _concatToken[token]
	} else {
		concat.Fprogress = -1
	}

	respon, err := json.Marshal(concat)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_JSON2BODY_ERROR)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(respon))

	return
}

// ConcatVideo 合并不同的视频 仅处理视频数量低于5个的请求
func ConcatVideo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, GET_BODY_ERROR)
		return
	}

	var concat Concat

	err = json.Unmarshal(content, &concat)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_BODY2JSON_ERROR)
		return
	}

	concat.FisRandom = false

	if len(concat.Finput) > MAX_VIDEO_LEN {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, TOO_MANY_CONCAT_REQ_ERROR)
		return
	}

	if concat.Foutput == "" {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, OUTPUT_EMPTY_ERROR)
		return
	}
	_tokenChan := make(chan string)

	go func() {
		token := getToken(FPS_TOKEN_SIZE)
		_tokenChan <- token

		_originLong := 0
		for _, v := range concat.Finput {
			time.Sleep(time.Duration(1) * time.Second)

			_o, err := _getVideoLength(v)
			if err != nil {
				_o = "00:00:01"
			}
			_originLong += _paserDurl(_o)
		}

		_concatToken := getConcatToken()
		_concatToken[token] = 0

		command := _concatCmd(concat.Finput, concat.FisRandom)

		command[0] += concat.Foutput

		cmds := []string{command[0]}

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
						_concatToken[token] = (float32(_dualTime) / float32(_originLong))
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

	var concatRespon ConcatResponse

	concatRespon.Ftoken = <-_tokenChan

	respon, err := json.Marshal(concatRespon)
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

// MultiConcatVideo 合并大量的不同的视频
func MultiConcatVideo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, GET_BODY_ERROR)
		return
	}

	var concat Concat

	err = json.Unmarshal(content, &concat)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_BODY2JSON_ERROR)
		return
	}

	concat.FisRandom = false

	if concat.Foutput == "" {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, OUTPUT_EMPTY_ERROR)
		return
	}
	_tokenChan := make(chan string)

	go func() {
		token := getToken(FPS_TOKEN_SIZE)
		_tokenChan <- token

		_originLong := 0
		for _, v := range concat.Finput {
			time.Sleep(time.Duration(1) * time.Second)

			_o, err := _getVideoLength(v)
			if err != nil {
				_o = "00:00:01"
			}
			_originLong += _paserDurl(_o)
		}

		_concatToken := getConcatToken()
		_concatToken[token] = 0

		var _concatName []string
		var cmds []string
		_concatName = concat.Finput
		//  开始合并
		for {
			command := _concatCmd(_concatName, concat.FisRandom)
			if len(command) <= 1 {
				cmds = append(cmds, command[0]+concat.Foutput)
				break
			}

			_concatName = _concatName[:0]
			for _, c := range command {
				tFile, err := ioutil.TempFile("", "video_")
				if err != nil {
					w.WriteHeader(ERROR)
					fmt.Fprintf(w, err.Error())
					return
				}

				c += tFile.Name()
				cmds = append(cmds, c+".mp4")
				_concatName = append(_concatName, tFile.Name()+".mp4")
			}

		}

		// command += concat.Foutput

		// cmds := []string{command}
		fmt.Println(cmds)
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
						_concatToken[token] = (float32(_dualTime) / float32(_originLong))
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

	var concatRespon ConcatResponse

	concatRespon.Ftoken = <-_tokenChan

	respon, err := json.Marshal(concatRespon)
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

func _concatCmd(video []string, isRadom bool) []string {

	if len(video) <= MAX_VIDEO_LEN {
		command := ""
		var _video []string
		if isRadom {
			_dict := make(map[int]int)
			for {
				index := rand.Intn(len(video))
				if _, ok := _dict[index]; !ok {
					_dict[index] = 0
					_video = append(_video, video[index])
				}

				if len(_video) >= len(video) {
					break
				}
			}
		} else {
			_video = video
		}

		for _, v := range video {
			command += ("-i " + v + " ")
		}

		command += " -y -filter_complex "

		_map := ""
		for i := range video {
			_map += fmt.Sprintf("[%d:v:0][%d:a:0]", i, i)
		}

		command += _map

		command = fmt.Sprintf("%sconcat=n=%d:v=1:a=1[v][a]", command, len(video))
		command += " -map [v] -map [a] "
		return []string{command}
	}

	_index := 0
	var _video []string
	needStop := false
	for {
		if needStop {
			return _video
		}

		if (_index + MAX_VIDEO_LEN) < len(video) {
			_tvideo := video[_index : _index+MAX_VIDEO_LEN]
			_video = append(_video, _concatCmd(_tvideo, isRadom)...)
			_index += MAX_VIDEO_LEN
		} else {
			_tvideo := video[_index : len(video)-1]
			_video = append(_video, _concatCmd(_tvideo, isRadom)...)
			needStop = true
		}
	}
}
