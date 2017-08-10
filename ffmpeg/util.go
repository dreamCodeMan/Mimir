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

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// getFpsToken 获取fps帧率
func getFpsToken() map[string]float32 {
	if _fps_token == nil {
		_fps_token = make(map[string]float32)
	}

	return _fps_token
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
				return vt[:strings.Index(field[1], ",")], nil
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
	fmt.Println("准备执行命令:" + command)
	cmds := strings.Fields(command)
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
