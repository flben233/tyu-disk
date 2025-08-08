package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/oneclickvirt/fio"
	"github.com/shirou/gopsutil/disk"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type DiskResult struct {
	Name  string
	Read  float32
	Write float32
}

//go:embed fio-io-test.ini
var fioArgs string

const (
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Reset  = "\033[0m"
)

func testing(ctx context.Context) {
	go func() {
		i := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
				switch i % 4 {
				case 0:
					fmt.Printf("\rTesting... |")
				case 1:
					fmt.Printf("\rTesting... /")
				case 2:
					fmt.Printf("\rTesting... -")
				case 3:
					fmt.Printf("\rTesting... \\")
				}
				i++
			}
			time.Sleep(350 * time.Millisecond)
		}
	}()
}

func diskTest(fioCmd string) {
	// Prepare the FIO command and arguments
	eng := "libaio"
	if runtime.GOOS == "windows" {
		eng = "windowsaio"
	}
	tempFile, err := os.CreateTemp("", "args.fio")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tempFile.Name())
	_, err = io.WriteString(tempFile, fioArgs)
	if err != nil {
		panic(err)
	}
	usage, err := disk.Usage("./")
	if err != nil {
		panic(err)
	}
	size := "1g"
	if usage.Free < 1*1024*1024*1024 {
		size = strconv.Itoa(int(usage.Free/2/1024/1024)) + "m"
	}

	// Start testing
	ctx, cancel := context.WithCancel(context.Background())
	testing(ctx)
	cmd := exec.Command(fioCmd, tempFile.Name(), "--ioengine="+eng, "--size="+size, "--output-format=json")
	result, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	cancel()
	resultStr := string(result)
	index := strings.Index(resultStr, "{")
	var fioResult map[string]interface{}
	err = json.Unmarshal([]byte(resultStr[index:]), &fioResult)
	if err != nil {
		panic(err)
	}

	// Parse the result
	tests := []string{"SEQ1MQ8T1", "SEQ1MQ1T1", "RND4KQ32T1", "RND4KQ1T1"}
	finalResult := make(map[string]*DiskResult)
	for _, job := range fioResult["jobs"].([]interface{}) {
		jobMap := job.(map[string]interface{})
		name := jobMap["jobname"].(string)
		for _, test := range tests {
			if strings.Contains(name, test) {
				if _, exists := finalResult[test]; !exists {
					finalResult[test] = &DiskResult{Name: test}
				}
				if strings.Contains(name, "read") {
					finalResult[test].Read = float32(jobMap["read"].(map[string]interface{})["bw"].(float64)) / 1024
				} else if strings.Contains(name, "write") {
					finalResult[test].Write = float32(jobMap["write"].(map[string]interface{})["bw"].(float64)) / 1024
				}
				break
			}
		}
	}
	for _, test := range tests {
		fmt.Printf("\r%s%-32s%s %s%-32.2f %-11.2f%s\n", Yellow, finalResult[test].Name, Reset, Blue, finalResult[test].Read, finalResult[test].Write, Reset)
	}
}

func main() {
	fioCmd, tmpFile, err := fio.GetFIO()
	if err != nil {
		panic(err)
	}
	defer fio.CleanFio(tmpFile)
	fmt.Println("-------------------------------- TyuDiskMark --------------------------------")
	fmt.Println("Developer             : ShirakawaTyu")
	fmt.Println("Last Maintaining      : 2025-08-08")
	fmt.Println("GitHub                : github.com/shirakawatyu/tyu-disk")
	fmt.Println("-----------------------------------------------------------------------------")
	fmt.Printf("%-32s %-32s %-25s\n", "Test", "Read(MB/s)", "Write(MB/s)")
	diskTest(fioCmd)
	fmt.Println("-----------------------------------------------------------------------------")
	fmt.Println("系统时间：", time.Now().Format("2006-01-02 15:04:05 MST"))
	fmt.Println("北京时间：", time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05"), "CST")
	fmt.Println("-----------------------------------------------------------------------------")
}
