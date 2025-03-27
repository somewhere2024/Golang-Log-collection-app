/*
基于golang实现的日志收集小程序
*/

package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Reader interface {
	Read(readchan chan []byte)
}
type Writer interface {
	Write(writechan chan Message)
}

type Logprocess struct {
	readchan  chan []byte
	writechan chan Message
	read      Reader
	write     Writer
}
type ReaderFromFile struct {
	path string
}
type WriteToInFluxdb struct {
	influxdb string
}

type Message struct {
	TimeLocal                        time.Time
	ByteSend                         int //传输的字节大小
	Path, Method, Status, RemoteAddr string
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
func (l *WriteToInFluxdb) Write(writechan chan Message) {
	for data := range writechan {
		fmt.Println(data)
	}
}

func (l *Logprocess) Process() {
	// 192.168.1.1 - - [27/Mar/2025:10:15:55 +0000] "GET /index.html HTTP/1.1" 200 1024 "http://example.com" "Mozilla/5.0 (Windows NT 10.0; Win64; x64)" "203.0.113.195"
	regex := `^(?P<remote_addr>\S+) - (?P<remote_user>\S*) \[(?P<time_local>[^\]]+)\] "(?P<request>[^"]+)" (?P<status>\d{3}) (?P<body_bytes_sent>\d+) "(?P<http_referer>[^"]*)" "(?P<http_user_agent>[^"]*)" "(?P<http_x_forwarded_for>[^"]*)"`
	r := regexp.MustCompile(regex)

	Mytimezone, _ := time.LoadLocation("Asia/Shanghai")

	for data := range l.readchan {
		ret := r.FindStringSubmatch(string(data))
		if len(ret) == 0 {
			log.Println("FindStringSubmatch fail", string(data))
			continue
		}
		msg := Message{}
		logtime, err := time.ParseInLocation("02/Jan/2006:15:04:05 +0000", ret[2], Mytimezone)
		if err != nil {
			log.Println("time.ParseInLocation fail", err.Error(), ret[2])
			continue
		}
		msg.TimeLocal = logtime
		bytesend, _ := strconv.Atoi(ret[5])
		msg.ByteSend = bytesend

		//匹配path
		//"GET /index.html HTTP/1.1" ret[3]
		req := strings.Split(ret[3], " ")
		if len(req) != 3 {
			log.Println("req fail", ret[3])
		}
		method := req[0]
		msg.Method = method
		url, err := url.Parse(req[1])
		if err != nil {
			log.Println("url.Parse fail", err.Error(), req[1])
			continue
		}
		msg.Path = url.Path
		msg.Status = req[4]
		l.writechan <- msg
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
		writechan: make(chan Message),
		read:      r,
		write:     w,
	}

	go l.read.Read(l.readchan)
	go l.Process()
	go l.write.Write(l.writechan)
	time.Sleep(time.Second * 30)
}
