package main

import (
    //"fmt"
    unc "github.com/hq-cml/unicorn-go/unicorn"
    "time"
)

//unicorn实例结构体
type UnicornInstance struct {
    qps uint32 //规定每秒的请求量
    timeout time.Duration //规定的每个请求最大延迟
    duration time.Duration //持续探测访问持续时间
    concurrency uint32 //并发量，这个值是根据timeout和qps算出来的
    stopSign chan byte //停止信号接收通道
    status unc.UncStatus //当前状态
    resultCh chan *unc.CallResult //保存调用结果的通道
    plugin unc.Plugin //插件

    //pool unc.WorkerPool

    //cancelSing byte// 取消发送后续结果的信号。
    //endSign     chan uint64          // 完结信号的传递通道，同时被用于传递调用执行计数。
    //callCount   uint64               // 调用执行计数。
}

func main() {
    //logger := lib.Logger{}
    unc.Logger.Info("HAHA")
}
