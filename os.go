package main

import (
	"runtime"
	"strconv"
	"strings"
)

type OS struct {
	Name  string `json:"name"`
	Arch  string `json:"arch"`
	Osbit int    `json:"bit"`
}

func GetOS() (*OS, error) {
	o := new(OS)

	const PtrSize = 32 << uintptr(^uintptr(0)>>63)

	o.Name = runtime.GOOS
	o.Arch = runtime.GOARCH
	o.Osbit = strconv.IntSize

	return o, nil
}

func DownFFmpeg(o *OS) error {
	switch strings.ToLower(o.Name) {
	case "darwin":
	case "linux":
	case "windows":
	default:

	}

	return nil
}

func CheckFFmpeg() error {
	return nil
}
