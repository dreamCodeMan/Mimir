package ffmpeg

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os/exec"
	"strings"
	"time"

	"github.com/kr/pty"
)

// _fps_token  保存fps帧率
var _fps_token map[string]float32

// _ratio_token 保存分辨率操作
var _ratio_token map[string]float32

// _concat_token 保存合并操作
var _concat_token map[string]float32

// _audio_token 保存合并操作
var _audio_token map[string]float32

// _logo_token 保存Logo操作
var _logo_token map[string]float32

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// getFpsToken 获取fps帧率
func getFpsToken() map[string]float32 {
	if _fps_token == nil {
		_fps_token = make(map[string]float32)
	}

	return _fps_token
}

// getRatioToken 获取分辨率
func getRatioToken() map[string]float32 {
	if _ratio_token == nil {
		_ratio_token = make(map[string]float32)
	}

	return _ratio_token
}

// getConcatToken 获取合并进度
func getConcatToken() map[string]float32 {
	if _concat_token == nil {
		_concat_token = make(map[string]float32)
	}

	return _concat_token
}

// getLogoToken 获取合并进度
func getLogoToken() map[string]float32 {
	if _logo_token == nil {
		_logo_token = make(map[string]float32)
	}

	return _logo_token
}

// getAudioToken 获取合并进度
func getAudioToken() map[string]float32 {
	if _audio_token == nil {
		_audio_token = make(map[string]float32)
	}

	return _audio_token
}

func getToken(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// _getVideoRatio 获取视频分辨率
func _getVideoRatio(input string) (string, error) {
	command := "-i " + input

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
		// fmt.Println("-=-=")
		isExit := false
		select {
		case e := <-_errorChan:
			return "", e
		case d := <-_chan:
			// fmt.Println(d)
			if strings.Contains(d, "Stream ") {
				_split := strings.Split(d, "Stream")
				field := strings.Split(_split[1], "[")
				if len(field) < 2 {
					return "", errors.New("Get Ratio Error!")
				}

				vt := strings.Split(field[0], ",")

				return strings.TrimSpace(vt[len(vt)-1]), nil
			}
		case <-_exit:
			isExit = true
		}

		if isExit {
			break
		}
	}

	return "", nil
}

// _get_video_length 获取视频总长度
func _getVideoLength(input string) (string, error) {

	command := "-i " + input

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
		// fmt.Println("-=-=")
		isExit := false
		select {
		case e := <-_errorChan:
			return "", e
		case d := <-_chan:
			// fmt.Println(d)
			if strings.Contains(d, "Duration: ") {
				_split := strings.Split(d, "Duration")
				field := strings.Fields(_split[1])
				vt := field[1]
				// fmt.Println("GET LENGTH : ", vt)
				if strings.Contains(vt, ",") {
					return vt[:strings.Index(field[1], ",")], nil
				}

				return vt, nil
			}
		case <-_exit:
			isExit = true
		}

		if isExit {
			break
		}
	}

	return "", nil
}

// _exec 执行指定的命令 _data_chan 用于返回命令执行的输出
func _exec(command string, _dataChan chan chan string) error {
	// 接受一个用于数据同步的chan
	_chan := <-_dataChan

	_needExit := make(chan int)
	cmds := strings.Fields(command)

	cmds = _paddingCMD(cmds)
	fmt.Println("准备执行命令:")
	fmt.Println(cmds)

	cmd := exec.Command("ffmpeg", cmds...)

	f, err := pty.Start(cmd)
	if err != nil {
		return err
	}

	tFile, err := ioutil.TempFile("", "Mimir")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(tFile.Name())

	defer func() {
		tFile.Close()
	}()

	var out []byte

	go func() {
		guard := 0 //guard哨兵用于判断当前命令是否发生变化
		isExit := false

		for {

			select {
			case <-_needExit:
				isExit = true
			default:
				out, _ = ioutil.ReadFile(tFile.Name())
				l := len(strings.TrimSpace(string(out)))
				if guard < l {
					str := string(out)[guard:l]
					// fmt.Println(str)
					_chan <- str
					guard = l
				}
			}

			if isExit {
				return
			}

		}
	}()

	io.Copy(tFile, f)

	err = cmd.Wait()
	if err != nil {
		return err
	}

	close(_chan)
	_needExit <- 1
	fmt.Println("exec finish")
	return nil
}

// _paddingCMD 过滤命令,某些参数需要添加空格
func _paddingCMD(cmd []string) []string {
	var _c []string
	for _, c := range cmd {
		if strings.Contains(c, "<SPACE>") {
			c = strings.Replace(c, "<SPACE>", " ", -1)
		}

		_c = append(_c, c)
	}

	return _c
}
