package probe

import (
    "syscall"
    "os"
    "time"
    "math"
    "strconv"
    "errors"
    "path/filepath"
)

var KB = uint64(1024)

type DiskUsage struct {
    Free uint64 `json:"free"`
    Available uint64 `json:"available"`
    Size uint64 `json:"size"`
    Used uint64 `json:"used"`
    Usage string `json:"usage"`
    Dir uint64 `json:"dir"`
    path string
    refresh uint64
}

func NewDiskUsage(volumePath string, refresh uint64) (*DiskUsage, error) {
    _, err := os.Stat(volumePath)
    if os.IsNotExist(err) {
        return nil, errors.New("Path "+volumePath+"does not exists")
    }
	return &DiskUsage{path: volumePath, refresh: refresh}, nil
}

func (usage *DiskUsage) Update() {
	var stat syscall.Statfs_t
	syscall.Statfs(usage.path, &stat)
	usage.Free = (stat.Bfree * uint64(stat.Bsize)) /KB
	usage.Available = (stat.Bavail * uint64(stat.Bsize)) /KB
	usage.Size = (stat.Blocks * uint64(stat.Bsize)) /KB
	usage.Used = usage.Size - usage.Free
	usage.Usage = percent(float64(usage.Used), float64(usage.Size))
	usage.Dir = dirSize(usage.path)
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
    filepath.Walk(path, func(path string, file os.FileInfo, err error) error {
        if !file.IsDir() {
            dirSize += uint64(file.Size())
        }
        return nil
    })
    return uint64(dirSize) /KB
}

func percent(val float64, limit float64) string {
    return strconv.FormatInt(int64(math.Abs(val * 100 / limit)), 10)+"%%"
}
