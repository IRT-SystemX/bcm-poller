package probe

import (
	"log"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

var KB = uint64(1024)

type DiskUsage struct {
	Free      uint64 `json:"free"`
	Available uint64 `json:"available"`
	Size      uint64 `json:"size"`
	Used      uint64 `json:"used"`
	Usage     uint64 `json:"usage"`
	Dir       uint64 `json:"dir"`
	path      string
	refresh   uint64
}

func NewDiskUsage(volumePath string, refresh uint64) *DiskUsage {
	_, err := os.Stat(volumePath)
	if os.IsNotExist(err) {
		err := os.MkdirAll(volumePath, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	return &DiskUsage{path: volumePath, refresh: refresh}
}

func (usage *DiskUsage) Update() {
	var stat syscall.Statfs_t
	err := syscall.Statfs(usage.path, &stat)
	if err != nil {
		log.Println("Error disk: ", err)
	} else {
		usage.Free = (stat.Bfree * uint64(stat.Bsize)) / KB
		usage.Available = (stat.Bavail * uint64(stat.Bsize)) / KB
		usage.Size = (stat.Blocks * uint64(stat.Bsize)) / KB
		usage.Used = usage.Size - usage.Free
		usage.Usage = uint64(math.Abs(float64(usage.Used) * 100 / float64(usage.Size)))
		usage.Dir = dirSize(usage.path)
	}
}

func (usage *DiskUsage) Start() {
	go func() {
		for {
			usage.Update()
			time.Sleep(time.Duration(usage.refresh) * time.Second)
		}
	}()
}

func dirSize(path string) uint64 {
	var dirSize uint64 = 0
	err := filepath.Walk(path, func(path string, file os.FileInfo, err error) error {
		if err == nil {
			if !file.IsDir() {
				//log.Printf("%s: %d", file.Name(), file.Size())
				dirSize += uint64(file.Size())
			}
		}
		return nil
	})
	if err != nil {
		log.Println("Error dirSize: ", err)
	}
	return uint64(dirSize) / KB
}
