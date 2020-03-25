package main

import (
	"fmt"
	"sync"
	"time"
)

/*
	常用的性能指标:
	PV(Page View):用户通过浏览器访问页面，对应用服务器产生的每一次请求，记为一个 PV
	-->
		性能测试环境下将这个概念做了延伸:系统真实处理的一个请求，视为一个 PV（PV的概念也适用于接口）
		可通过监控埋点或者统计访问日志统计
		PeakPV，指一天中 PV数达到的高峰PV值--->日PV可以理解为每天的总访问量

	QPS/TPS:系统每秒能处理的请求/事务的数量（吞吐量）.
	在web应用中更关注的是web应用每秒能处理的request数量
	QPS（TPS）= 并发数/平均响应时间
	可通过访问日志统计对应时间的PV量除以对应时间得到，一般经常统计的是高峰期PV对应的QPS

	ResponseTime响应时间（时延）
	从客户端发一个请求开始计时，到客户端接收到从服务器端返回的响应结果结束所经历的时间
	响应时间由请求发送时间、网络传输时间和服务器处理时间三部分组成

	LOAD负载
	系统平均负载，被定义为在特定时间间隔内运行队列中的平均进程数
	如果一个进程满足以下条件则其就会位于运行队列中：
	（1）没有在等待 I/O操作的结果
	（2）没有主动进入等待状态（也就是没有调用wait或sleep等）
	（3）没有被停止
	这个负载值比较理想的指标值是cpu个数*核数*0.7，若长期超过这个值就需要对系统进行警惕

	CPU 资源
	指应用服务系统的 CPU 资源占用率

	关于JAVA（JVM GC和FullGC）--->golang
	FGC会进行完全的垃圾清理，会使应用运行得很慢，应该尽量避免FGC
	-->通过设置合适的JVM参数和GC策略来避免FGC
	通常监控的指标有GC次数和响应时间

	!!!联系和区别:
	1.容量预测
	在上线前需要测试下能接收用户多大的访问量,也就是评估出最大的日PV到来时，系统是否能支撑。
	这里如何先通过现有数据（暂时无需做极限压测）总结出了高峰QPS和日pv的关系：
	所需数据：
	（1）每台机器的QPS
	（2）每天的访问量(日PV)
	（3）日PV的实际访问分布（具体情况不同，如每天80%的访问集中在20%的时间里，下面看这个来算；或者全算在白天即每天的访问集中在50%的时间里）
	 ---> (日PV量 * 80%) / (24*60*60*20%) / 每台机器的QPS = 服务需要部署多少台机器

	测试工具直接按每秒发多少个请求(qps)来测试：此时看CPU负载，时延（请求量不降，当处理不过来时会造成不断增加）
	定一个qps，看看此时系统的处理能力

	并发量与时延：
		无关系的角度：并发量变化的情况下时延可能不会变（系统能处理过来），时延可能会因内部服务变化而变化
		有关系的角度：并发量越高可能会导致时延增大

	要知道一点：吞吐量与时延不是直接影响的关系（吞吐量高不代表时延低）
*/

// RunFunc run func 要测试的接口
type RunFunc func() error

// StatPV pv statistics
type StatPV struct {
	ReqLock     sync.Mutex
	ConcWG      sync.WaitGroup
	ConcChan    chan int
	ReqNum      int
	QPS         int
	TimeConsume map[string]int
	AvrConsume  float64
	MinTime     int
	MaxTime     int
	Total       int64
	Fail        int64
	FunV        RunFunc
}

func (sp *StatPV) outPut(elapsedTime int) {
	fmt.Printf("elapsedTime:\t%d\n", elapsedTime) //总耗时
	fmt.Printf("0_10ms:\t%d\n", sp.TimeConsume["0_10ms"])
	fmt.Printf("10_30ms:\t%d\n", sp.TimeConsume["10_30ms"])
	fmt.Printf("30_50ms:\t%d\n", sp.TimeConsume["30_50ms"])
	fmt.Printf("50_100ms:\t%d\n", sp.TimeConsume["50_100ms"])
	fmt.Printf("100_200ms:\t%d\n", sp.TimeConsume["100_200ms"])
	fmt.Printf("200_500ms:\t%d\n", sp.TimeConsume["200_500ms"])
	fmt.Printf("500_+00ms:\t%d\n", sp.TimeConsume["500_+00ms"])
	fmt.Printf("minTime:\t%dms\n", sp.MinTime)
	fmt.Printf("maxTime:\t%dms\n", sp.MaxTime)
	fmt.Printf("total req:\t%d\n", sp.Total)
	fmt.Printf("fail req:\t%d\n", sp.Fail)
	if elapsedTime < 1 {
		elapsedTime = 1
	}
	pv := sp.Total * 1000 / int64(elapsedTime)
	fmt.Printf("pv:\t%d\n", pv)
}

func (sp *StatPV) runFunc() {
	begin := time.Now()
	err := sp.FunV()
	elapsedTime := int(time.Now().Sub(begin) / 1000000)
	sp.ReqLock.Lock()
	defer sp.ReqLock.Unlock()
	if err != nil {
		//接口处理失败（包括可能的超时）
		sp.Fail += int64(1)
	}
	sp.AvrConsume = (sp.AvrConsume*float64(sp.Total) + float64(elapsedTime)) / float64(sp.Total+int64(1))
	sp.Total += int64(1)
	if elapsedTime > sp.MaxTime {
		sp.MaxTime = elapsedTime
	}
	if elapsedTime < sp.MinTime {
		sp.MinTime = elapsedTime
	}
	if elapsedTime < 10 {
		sp.TimeConsume["0_10ms"]++
	} else if elapsedTime < 30 {
		sp.TimeConsume["10_30ms"]++
	} else if elapsedTime < 50 {
		sp.TimeConsume["30_50ms"]++
	} else if elapsedTime < 100 {
		sp.TimeConsume["50_100ms"]++
	} else if elapsedTime < 200 {
		sp.TimeConsume["100_200ms"]++
	} else if elapsedTime < 500 {
		sp.TimeConsume["200_500ms"]++
	} else {
		sp.TimeConsume["500_+00ms"]++
	}
	<-sp.ConcChan //释放连接池
	sp.ConcWG.Done()
}

func (sp *StatPV) totalLimit() {
	begin := time.Now()
	qpm := float64(0)
	if sp.QPS > 0 {
		qpm = float64(sp.QPS) / float64(1000) //每毫秒处理多少
	}
	for index := 0; index < sp.ReqNum; {
		if sp.QPS > 0 {
			elapsedTime := time.Now().Sub(begin) / 1000000 //当前耗时**毫秒
			if qpm*float64(elapsedTime) < float64(index) { //qpm*耗时**秒=按设置的qps应该处理到的量，超过则限制
				time.Sleep(time.Microsecond * 100)
				continue
			}
		}
		/*
			关于并发数的设定：
				先看单次请求时延N ms，相当于每秒处理 1000/N 次，conn可设置为（QPS / 次数）
				这种方式可粗略的在1秒内分批发送，避免同时上来的情况
		*/
		sp.ConcChan <- 1 //通过连接数缓存控制并发数（若并发数=qps，相当于是同时上来）
		sp.ConcWG.Add(1)
		go sp.runFunc()
		if (index+1)%5000 == 0 {
			elapsedTime := int(time.Now().Sub(begin) / 1000000) //每5000次请求输出一次
			sp.outPut(elapsedTime)
		}
		index++
	}
	sp.ConcWG.Wait()
	elapsedTime := int(time.Now().Sub(begin) / 1000000)
	sp.outPut(elapsedTime)
}

func (sp *StatPV) rateLimit() {

}

// Start run
func (sp *StatPV) Start() {
	sp.totalLimit()
}

//NewSP new stat pv
/*
	（人工）Qps设置为0，则根据并发数来控制（随着CONN的增多，系统时延可能会变大）
*/
func NewSP(funcv RunFunc, qps, reqNum, conc int) *StatPV {
	//时延分布
	timeconsume := make(map[string]int)
	timeconsume["0_10ms"] = 0
	timeconsume["10_30ms"] = 0
	timeconsume["30_50ms"] = 0
	timeconsume["50_100ms"] = 0
	timeconsume["100_200ms"] = 0
	timeconsume["200_500ms"] = 0
	timeconsume["500_+00ms"] = 0
	if conc < 1 {
		conc = 1
	}
	concChan := make(chan int, conc) //连接数缓存
	sp := StatPV{
		ConcChan:    concChan,
		ReqNum:      reqNum,
		QPS:         qps,
		TimeConsume: timeconsume,
		FunV:        funcv}

	return &sp
}
