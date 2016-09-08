package main

import (
    //"fmt"
    unicorn "github.com/hq-cml/unicorn-go/unicorn"
    "time"
    "errors"
    "math"
)

//Unicorn接口
type UnicornIntfs interface {
    Start()                //启动unicorn
    Stop() (uint64, bool)  //返回值1表示停止时已完成请求数，返回值2表示是否成功停止
    Status() unicorn.UncStatus //获得unicorn当前状态
}

//Unicorn接口的实现类型
type Unicorn struct {
    qps         uint32                   //规定每秒的请求量
    timeout     time.Duration            //规定的每个请求最大延迟
    duration    time.Duration            //持续探测访问持续时间
    concurrency uint32                   //并发量，这个值是根据timeout和qps算出来的
    stopSign    chan byte                //停止信号接收通道
    status      unicorn.UncStatus        //当前状态
    resultChan  chan *unicorn.CallResult //保存调用结果的通道
    plugin      unicorn.Plugin           //插件
    pool        unicorn.WorkerPool       //协程池
    cancelSign  byte                     //取消发送后续结果的信号标记。

                                         //endSign     chan uint64          // 完结信号的传递通道，同时被用于传递调用执行计数。
                                         //callCount   uint64               // 调用执行计数。
}

//初始化Unicorn，几件重要的事情
//1. 计算出合适的并发量
//2. 实例化线程池
func (unc *Unicorn) init() error {
    unicorn.Logger.Info("Begin Init Unicorn")

    //计算并发量concurrency ≈ 规定响应超时时间 / 发送间隔时间
    var conc int64 = int(unc.timeout) / int64(1e9/ unc.qps) + 1
    if conc > math.MaxInt32 {
        conc = math.MaxInt32
    }
    unc.concurrency = uint32(conc)

    //实例化线程池
    wp, err := unicorn.NewWorkerPool(unc.concurrency)
    if err != nil {
        return err
    }
    unc.pool = wp

    return nil
}

//实例化Unicorn，惯例New函数
//返回的是UnicornIntfs的实现
func NewUnicorn(plugin unicorn.PluginIntfs, timeout time.Duration, qps uint32, duration time.Duration, resultChan chan *unicorn.CallResult) (UnicornIntfs, error) {
    unicorn.Logger.Info("Begin New Unicorn")

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
    unc := &Unicorn{
        plugin     : plugin,
        timeout    : timeout,
        qps        : qps,
        duration   : duration,
        stopSign   : make(chan byte, 1),
        cancelSign : 0,
        status     : unicorn.ORIGINAL,
        resultChan : resultChan,
    }

    //初始化
    if err := unc.init(); err != nil {
        return nil, err
    }
    return unc, nil
}

//*Unicorn实现Unicorn接口


//处理停止“信号”
func (unc * Unicorn) handleStopSign() {
    //信号标记变为1
    unc.cancelSign = 1
    unicorn.Logger.Info("handleStopSign. Closing result chan...")
    //关闭结果存储通道
    close(unc.resultChan)
}

//发送请求的主控制逻辑
//通过节流阀throttle控制发送请求的强度
//请求过程中不断检测stopSign，如果检测到，则将最终结果传入endSign
func (unc* Unicorn) genRequest(throttle <-chan time.Time, endSign chan<- uint64) {
    call_cnt := uint64(0)

Loop:
    //一个无限循环，只要满足条件，就发送请求
    for ;;call_cnt++ {
        //带default的select分支，是不会出现阻塞的
        select {
        case <- unc.stopSign:
            unc.handleStopSign()
            endSign <- call_cnt
            break Loop
        default:
        }

        //实际发送请求
        unc.sendRequest()

        //阻塞等待节流阀throttle信号
        if unc.qps > 0 {
            select {
            case <-throttle: //空转一次，进入下次循环，发送请求
            case <-unc.stopSign:
                unc.handleStopSign()
                endSign <- call_cnt
                break Loop
            }
        }
    }
}

func main() {
    //logger := lib.Logger{}
    unicorn.Logger.Info("HAHA")
}
