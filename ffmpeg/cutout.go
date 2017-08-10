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
	GET_VIDEO_LENGTH = "10008"
)

type CutOut struct {
	Ffmpeg FFMPEG `json:"video"`
	// Fnum 视频个数，默认5个
	Fnum int `json:"num"`
	// Finter 视频长度，默认3秒
	Finter string `json:"inter"`
}

type CutOutResponse struct {
	// Fname 截取后的文件路径
	Fname []string `json:"output"`
}

// VideoCutOut 随机截取视频
func VideoCutOut(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, GET_BODY_ERROR)
		return
	}

	var cutout CutOut

	err = json.Unmarshal(content, &cutout)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_BODY2JSON_ERROR)
		return
	}

	outDir := filepath.Dir(cutout.Ffmpeg.Finput)
	outDir += "/"

	if cutout.Fnum == 0 {
		cutout.Fnum = 5
	}

	if cutout.Finter == "" {
		cutout.Finter = "00:03"
	}

	_videoLength, err := _getVideoLength(cutout.Ffmpeg.Finput)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, GET_VIDEO_LENGTH)
		return
	}

	_timeList := _generateRandomLength(_videoLength, cutout.Fnum)
	var cutoutRespon CutOutResponse

	file := filepath.Base(cutout.Ffmpeg.Finput)
	name := strings.Split(file, ".")
	n := strings.Join(name[:len(name)-1], ".")
	// 替换帧间编码
	// -strict -2  -qscale 0 -intra keyoutput.mp4
	tmpFile := outDir + n + "_tmp_." + name[len(name)-1]
	command := "-i " + cutout.Ffmpeg.Finput + " "
	command += ("-y -strict -2  -qscale 0 -intra " + tmpFile)

	cmds := []string{command}
	_dataChan := make(chan chan string)
	_errorChan := make(chan error)
	_exit := make(chan int)

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
			case <-_chan:
			case <-_exit:
				isExit = true
			}

			if isExit {
				break
			}
		}
	}

	for index := 0; index < cutout.Fnum; index++ {

		cutout.Ffmpeg.Fouput = outDir + n + "_cut_" + strconv.Itoa(index) + "." + name[len(name)-1]
		cutoutRespon.Fname = append(cutoutRespon.Fname, cutout.Ffmpeg.Fouput)

		fmt.Println(cutout.Ffmpeg.Fouput)

		command := "-ss " + _timeList[index] + " -i " + tmpFile + " "
		command += ("-y " + " -t " + fmt.Sprintf("00:%s", cutout.Finter) + " -vcodec copy -acodec copy ")
		command += cutout.Ffmpeg.Fouput

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
				case <-_chan:
					// if strings.Contains(s, " bitrate=") {
					// 	_info := strings.Split(s, " bitrate=")
					// 	_time := strings.Split(_info[0], "time=")
					// 	// _dualTime := _paserDurl(_time[1])
					// 	// _logoToken[token] = (float32(_dualTime) / float32(_originLong))
					// }
				case <-_exit:
					isExit = true
				}

				if isExit {
					break
				}
			}
		}

	}

	// cutoutRespon.Fname =
	respon, err := json.Marshal(cutoutRespon)
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

func _generateTime(start string, inter int) string {
	_t := strings.Split(start, ":")
	_si, err := strconv.ParseFloat(_t[2], 64)
	if err != nil {
		_si = 0.0
	}

	_mi, _ := strconv.Atoi(_t[1])
	_hi, _ := strconv.Atoi(_t[0])
	_si += float64(inter)

	if _si >= 60.0 {
		_si = _si - 60.0
		_mi++
	}

	if _mi >= 60 {
		_mi = _mi - 60
		_hi++
	}

	return fmt.Sprintf("%02d:%02d:%.02f", _hi, _mi, _si)

}
