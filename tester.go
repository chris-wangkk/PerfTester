package main

import (
	"flag"
	"runtime"
)

var (
	reqType int //支持一个程序下选择多个请求
	qps     int //设定的qps值
	reqNum  int //发送总数
	conc    int //并发量
)

func TestMock() error {
	return nil
}

func main() {
	sp := NewSP(TestMock, qps, reqNum, conc)
	sp.Start()
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.IntVar(&reqType, "type", 0, "client request type...")
	flag.IntVar(&qps, "qps", 0, "qps")
	flag.IntVar(&reqNum, "reqNum", 1, "number of requests")
	flag.IntVar(&conc, "conc", 1, "number of client")
	flag.Parse()
}
