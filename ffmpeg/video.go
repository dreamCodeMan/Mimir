package ffmpeg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

const (
	ERROR                 = 501
	GET_BODY_ERROR        = "10001"
	PARSE_BODY2JSON_ERROR = "10002"
	CMD_EXEC_ERROR        = "10003"
	PARSE_JSON2BODY_ERROR = "10004"
)

type FFMPEG struct {
	// Finput 源文件
	Finput string `json:"input"`
	// Foutput 输出文件
	Fouput string `json:"output"`
}

type Shot struct {
	Ffmpeg FFMPEG `json:"video"`
	// Stype 截图类型 0:截取指定时间点 1:按间隔秒数截取 2:截取指定数量
	Stype int `json:"type"`
	// Svalue 类型值
	// 当类型为0时，此处按照hh:mm:ss 填写时间
	// 当类型为1时, 此处填写秒数
	Svalue string `json:"value"`

	// Simg 输出的图片类型
	// 当类型为0时，此处为输出的图片名称
	// 当类型为1时，按照以下格式赋值 图片前缀|类型后缀 e.g. img|jpg 以img为前缀的jpg类型图片
	Simg string `json:"outformat"`
	// Ssize 输出的图片尺寸
	Ssize string `json:"size"`
}

type ShotResponse struct {
	Img []string `json:"img"`
}

// VideoShot 视频截图
func VideoShot(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, GET_BODY_ERROR)
		return
	}

	var shot Shot

	err = json.Unmarshal(content, &shot)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_BODY2JSON_ERROR)
		return
	}

	var cmds []string
	outDir := filepath.Dir(shot.Ffmpeg.Finput)
	outDir += "/"
	shot.Simg = outDir + shot.Simg

	command := ""

	switch shot.Stype {
	case 0:
		command += "-ss "
		command += (shot.Svalue + " ")
		command += ("-i " + shot.Ffmpeg.Finput + " ")
		command += "-y -vframes 1 "
		command += shot.Simg

		cmds = append(cmds, command)
	case 1:
		command += ("-i " + shot.Ffmpeg.Finput + " ")
		command += ("-vf fps=fps=1/" + shot.Svalue + " ")
		oi := strings.Split(shot.Simg, "|")
		if len(oi) != 2 {
			oi[0] = "img"
			oi[1] = "png"
		}
		command += (oi[0] + "%03d." + oi[1])
		cmds = append(cmds, command)
	case 3:
		duration, err := _getVideoLength(shot.Ffmpeg.Finput)
		if err != nil {
			// 获取视频长度失败，默认只截取一张
			command += "-ss "
			command += (" 00:00:01 ")
			command += ("-i " + shot.Ffmpeg.Finput + " ")
			command += "-y -vframes 1 "
			oi := strings.Split(shot.Simg, "|")
			if len(oi) != 2 {
				oi[0] = "img"
				oi[1] = "png"
			}
			command += (oi[0] + "." + oi[1])
			cmds = append(cmds, command)
		} else {
			num, err := strconv.Atoi(shot.Svalue)
			if err != nil {
				num = 1
			}
			_durList := _generateRandomLength(duration, num)
			fmt.Println(_durList)
			for i, d := range _durList {
				command = ""
				command += "-ss "
				command += (d + " ")
				command += ("-i " + shot.Ffmpeg.Finput + " ")
				command += "-y -vframes 1 "
				oi := strings.Split(shot.Simg, "|")
				if len(oi) != 2 {
					oi[0] = "img"
					oi[1] = "png"
				}
				command += (oi[0] + strconv.Itoa(i) + "." + oi[1])
				cmds = append(cmds, command)
			}
		}

		if len(cmds) == 0 {
			command += "-ss "
			command += (" 00:00:01 ")
			command += ("-i " + shot.Ffmpeg.Finput + " ")
			command += "-y -vframes 1 "
			oi := strings.Split(shot.Simg, "|")
			if len(oi) != 2 {
				oi[0] = "img"
				oi[1] = "png"
			}
			command += (oi[0] + "." + oi[1])
			cmds = append(cmds, command)
		}

	}

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

	var sr ShotResponse
	_dict := make(map[string]string)
	// fmt.Println(len(cmds))
	for _, c := range cmds {
		_chan := make(chan string)
		_dataChan <- _chan

		isExit := false
		for {
			select {
			case s := <-_chan:
				switch shot.Stype {
				case 0:
					_dict[shot.Simg] = ""
				case 1:
					// fmt.Println(s)
					if strings.Contains(s, "frame=") {
						_st := strings.Split(s, "fps=")
						_i := strings.Fields(_st[0])
						_inx, err := strconv.Atoi(_i[1])
						if err != nil {
							_inx = 1
						}
						oi := strings.Split(shot.Simg, "|")
						if len(oi) != 2 {
							oi[0] = "img"
							oi[1] = "png"
						}
						_formate := oi[0] + "%03d." + oi[1]
						_dict[fmt.Sprintf(_formate, _inx)] = ""
					}
				case 3:
					imgList := strings.Fields(c)
					_dict[imgList[len(imgList)-1]] = ""
				}
			case e := <-_errorChan:
				w.WriteHeader(ERROR)
				fmt.Fprintf(w, CMD_EXEC_ERROR+e.Error())
				return
			case <-_exit:
				isExit = true
			}

			if isExit {
				break
			}
		}
	}

	for k := range _dict {
		sr.Img = append(sr.Img, k)
	}
	respon, err := json.Marshal(sr)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_JSON2BODY_ERROR)
		close(_dataChan)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(respon))

	fmt.Println("Video Operation Finish")
	return
}

// _generateRandomLength 获取随机的时间点 duration 总的视频长度 hh:mm:ss格式 num 生成的随机时间点个数
func _generateRandomLength(duration string, num int) []string {
	var ds []string
	duartions := strings.Split(duration, ":")
	if len(duartions) != 3 {

	} else {
		// 获取hour
		for index := 0; index < num; index++ {

			_time := ""
			hour, err := strconv.Atoi(duartions[0])
			if err != nil {
				hour = 0
			}

			if hour > 1 {
				_rhour := rand.Intn(hour)
				_time += fmt.Sprintf("%02d:", _rhour)
			} else {
				_time += "00:"
			}

			min, err := strconv.Atoi(duartions[1])
			if err != nil {
				min = 0
			}

			if min > 1 {
				_rmin := rand.Intn(min)
				_time += fmt.Sprintf("%02d:", _rmin)
			} else {
				_time += "00:"
			}

			sec, err := strconv.ParseFloat(duartions[2], 64)
			if err != nil {
				sec = 0
			}

			if sec > 1 {
				_rsec := rand.Float32() * 60.0
				_time += fmt.Sprintf("%02.02g", _rsec)
			} else {
				_time += "00:"
			}

			ds = append(ds, _time)
		}

	}
	return ds
}

func populateStdin(file []byte) func(io.WriteCloser) {
	return func(stdin io.WriteCloser) {
		defer stdin.Close()
		io.Copy(stdin, bytes.NewReader(file))
	}
}
