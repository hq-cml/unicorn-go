package main

import (
    unicorn "github.com/hq-cml/unicorn-go/unicorn"
    "time"
    "errors"
    "math"
    "fmt"
    "bytes"
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
    plugin      unicorn.PluginIntfs      //插件
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

//兜底的错误处理，以defer的形式存在
func (unc *Unicorn) handleError() {
    if p := recover(); p != nil {
        var buff bytes.Buffer
        buff.WriteString("A Painic! (")
        err, ok := interface{}(p).(error) //类型转换 + 断言
        if ok { //断言成功
            buff.WriteString("error :" + err.Error())
        } else {
            buff.WriteString("clue :" + fmt.Sprintf("%v", p))
        }
        buff.WriteString(")")
        msg := buff.String()
        unicorn.Logger.Info(msg)

        //填充结果
        result = &unicorn.CallResult{
            Id     : -1,
            Code   : unicorn.RESULT_CODE_FATAL_CALL,
            Msg    : msg,
        }
        //结果存入通道
        unc.saveResult(result)
    }
}

//实际发送请求的逻辑
//既然是golang，很自然的应该想到这个逻辑应该是一个异步的goroutine
//但为了防止无限分配goroutine，所以结合worker_pool，实现goroutine总量控制
func (unc *Unicorn) sendRequest() {
    //Take和Return时机很重要，必须是主goroutine申请，子goroutine归还！
    //这个时机如果不正确，就无法起到控制goroutine的作用
    unc.pool.Take() //主goroutine申请派生

    //子goroutine
    go func() {
        //注册错误处理
        defer unc.handleError()

        //开始~
        raw_request := unc.plugin.GenReq()

        var result *unicorn.CallResult
        raw_response_chan := make(chan *unicorn.RawResponse, 1)
        //启动一个孙子goroutine
        go unc.interact(&raw_request, raw_response_chan)

        select {
        case raw_response := <-raw_response_chan:
            if raw_response.Err != nil {
                result = &unicorn.CallResult{
                    Id     : raw_response.Id,
                    Req    : raw_request,
                    Resp   : raw_response,
                    Code   : unicorn.RESULT_CODE_ERROR_CALL,
                    Msg    : raw_response.Err.Error(),
                    Elapse : raw_response.Elapse,
                }
            }else {
                result = unc.plugin.CheckResp(raw_request, *raw_response)
                result.Elapse = raw_response.Elapse
            }
        case <- time.After(unc.timeout):
            result = &unicorn.CallResult{
                Id     : raw_response.Id,
                Req    : raw_request,
                Code   : unicorn.RESULT_CODE_WARING_TIMEOUT,
                Msg    : fmt.Sprintf("Timeout! (expected: < %v)", unc.timeout),
            }
        }

        unc.saveResult(result) //结果存入通道
        unc.pool.Return() //子goroutine归还
    }()
}

//抽象的交互过程
func (unc *Unicorn)interact(rawReq *unicorn.RawReqest, rawRespChan chan<- *unicorn.RawResponse) {

}

func main() {
    //logger := lib.Logger{}
    unicorn.Logger.Info("HAHA")
}
