package main

import (
    //"fmt"
    unc "github.com/hq-cml/unicorn-go/unicorn"
    "time"
    "errors"
    "math"
)

//Unicorn接口
type UnicornIntfs interface {
    Start()                //启动unicorn
    Stop() (uint64, bool)  //返回值1表示停止时已完成请求数，返回值2表示是否成功停止
    Status() unc.UncStatus //获得unicorn当前状态
}

//Unicorn接口的实现类型
type Unicorn struct {
    qps             uint32                  //规定每秒的请求量
    timeout         time.Duration           //规定的每个请求最大延迟
    duration        time.Duration           //持续探测访问持续时间
    concurrency     uint32                  //并发量，这个值是根据timeout和qps算出来的
    stopSign        chan byte               //停止信号接收通道
    status          unc.UncStatus           //当前状态
    resultChan      chan *unc.CallResult    //保存调用结果的通道
    plugin          unc.Plugin              //插件
    pool            unc.WorkerPool          //协程池

    //cancelSign byte                  // 取消发送后续结果的信号。
    //endSign     chan uint64          // 完结信号的传递通道，同时被用于传递调用执行计数。
    //callCount   uint64               // 调用执行计数。
}

//初始化Unicorn，几件重要的事情
//1. 计算出合适的并发量
//2. 实例化线程池
func (u *Unicorn) init() error {
    unc.Logger.Info("Begin Init Unicorn")

    //计算并发量concurrency ≈ 规定响应超时时间 / 发送间隔时间
    var conc int64 = int(u.timeout) / int64(1e9/u.qps) + 1
    if conc > math.MaxInt32 {
        conc = math.MaxInt32
    }
    u.concurrency = uint32(conc)

    //实例化线程池
    wp, err := unc.NewWorkerPool(u.concurrency)
    if err != nil {
        return err
    }
    u.pool = wp

    return nil
}

//实例化Unicorn，惯例New函数
//返回的是UnicornIntfs的实现
func NewUnicorn(plugin unc.PluginIntfs, timeout time.Duration, qps uint32, duration time.Duration, resultChan chan *unc.CallResult) (UnicornIntfs, error) {
    unc.Logger.Info("Begin New Unicorn")

    //参数校验
    if plugin == nil {
        return nil, errors.New("Nil plugin")
    }
    if timeout == 0 {
        return nil, errors.New("Nil timeout")
    }
    if qps == nil {
        return nil, errors.New("Nil qps")
    }
    if duration == nil {
        return nil, errors.New("Nil duration")
    }
    if resultChan == nil {
        return nil, errors.New("Nil resultChan")
    }

    //创建instance
    ins := &Unicorn{
        plugin     : plugin,
        timeout    : timeout,
        qps        : qps,
        duration   : duration,
        stopSign   : make(chan byte, 1),
        //cancelSign: 0,
        status     : unc.ORIGINAL,
        resultChan : resultChan,
    }

    //初始化
    if err := ins.init(); err != nil {
        return nil, err
    }
    return ins, nil
}



func main() {
    //logger := lib.Logger{}
    unc.Logger.Info("HAHA")
}
