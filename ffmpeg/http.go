package ffmpeg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/kr/pty"
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

	command := ""

	switch shot.Stype {
	case 0:
		command += "-ss "
		command += (shot.Svalue + " ")
		command += ("-i " + shot.Ffmpeg.Finput + " ")
		command += "-vframes 1 "
		command += shot.Simg
	case 1:
		command += ("-i " + shot.Ffmpeg.Finput + " ")
		command += ("-vf fps=fps=1/" + shot.Svalue + " ")
		oi := strings.Split(shot.Simg, "|")
		if len(oi) != 2 {
			oi[0] = "img"
			oi[1] = "png"
		}
		command += (oi[0] + "%03d." + oi[1])
	}

	_data_chan := make(chan string)
	go func() {
		err = _exec(command, _data_chan)
		if err != nil {
			w.WriteHeader(ERROR)
			fmt.Fprintf(w, CMD_EXEC_ERROR+err.Error())
			close(_data_chan)
			return
		}
	}()

	var sr ShotResponse
	_dict := make(map[string]string)
	for s := range _data_chan {
		switch shot.Stype {
		case 0:
		case 1:
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
		}
	}

	fmt.Println("---END---")
	for k := range _dict {
		sr.Img = append(sr.Img, k)
	}
	respon, err := json.Marshal(sr)
	if err != nil {
		w.WriteHeader(ERROR)
		fmt.Fprintf(w, PARSE_JSON2BODY_ERROR)
		close(_data_chan)
		return
	}

	fmt.Fprintf(w, string(respon))
	return
}

// func _parse_respon(data string, s *Shot) *ShotResponse {
// 	sr := new(ShotResponse)
// 	return sr
// }

// _exec 执行指定的命令 _data_chan 用于返回命令执行的输出
func _exec(command string, _data_chan chan string) error {
	cmds := strings.Fields(command)
	// fmt.Println(cmds)
	cmd := exec.Command("ffmpeg", cmds...)

	f, err := pty.Start(cmd)
	if err != nil {
		return err
	}

	tFile, err := ioutil.TempFile("", "Mimir")
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		tFile.Close()
	}()

	var out []byte
	// var responData string
	go func() {
		guard := 0 //guard哨兵用于判断当前命令是否发生变化

		for {
			out, _ = ioutil.ReadFile(tFile.Name())
			l := len(strings.TrimSpace(string(out)))

			if guard < l {
				str := string(out)[guard:l]
				// fmt.Println(str)
				_data_chan <- str
				guard = l
			}
		}
	}()

	io.Copy(tFile, f)

	err = cmd.Wait()
	if err != nil {
		return err
	}

	close(_data_chan)
	return nil
}

func populateStdin(file []byte) func(io.WriteCloser) {
	return func(stdin io.WriteCloser) {
		defer stdin.Close()
		io.Copy(stdin, bytes.NewReader(file))
	}
}
