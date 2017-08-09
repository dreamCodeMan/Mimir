package ffmpeg

import (
	"math/rand"
	"time"
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
