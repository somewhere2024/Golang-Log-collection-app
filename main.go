/*
基于golang实现的日志收集小程序
*/

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type Reader interface {
	Read(readchan chan []byte)
}
type Writer interface {
	Write(writechan chan string)
}

type Logprocess struct {
	readchan  chan []byte
	writechan chan string
	read      Reader
	write     Writer
}
type ReaderFromFile struct {
	path string
}
type WriteToInFluxdb struct {
	influxdb string
}

func (l *ReaderFromFile) Read(readchan chan []byte) {
	f, err := os.Open(l.path)
	if err != nil {
		panic(fmt.Sprintln("open file error", err.Error()))
	}
	// 从文件末尾开始逐行读取文件内容
	f.Seek(0, 2)
	reads := bufio.NewReader(f)
	for {
		line, err := reads.ReadBytes('\n')
		if err == io.EOF {
			time.Sleep(time.Millisecond * 500)
			continue
		} else if err != nil {
			panic(fmt.Sprintf("read file error: %s", err.Error()))
		}
		readchan <- line[:len(line)-1] //消去最后的换行符
	}
}
func (l *WriteToInFluxdb) Write(writechan chan string) {
	for data := range writechan {
		fmt.Println(data)
	}
}

func (l *Logprocess) Process() {
	for data := range l.readchan {
		l.writechan <- strings.ToUpper(string(data))
	}
}

func main() {
	r := &ReaderFromFile{
		path: "./access.log",
	}
	w := &WriteToInFluxdb{
		influxdb: "username:password...",
	}
	l := &Logprocess{
		readchan:  make(chan []byte),
		writechan: make(chan string),
		read:      r,
		write:     w,
	}

	go l.read.Read(l.readchan)
	go l.Process()
	go l.write.Write(l.writechan)
	time.Sleep(time.Second * 30)
}
