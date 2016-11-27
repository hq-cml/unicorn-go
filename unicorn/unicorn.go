package unicorn

import (
    "github.com/hq-cml/unicorn-go/log"
    wp "github.com/hq-cml/unicorn-go/worker-pool"
    "math"
    "fmt"
    "time"
    "errors"
    "sync"
)

/*
 * 惯例New函数，实例化Unicorn，返回的是UnicornIntfs的实现
 */
func NewUnicorn(
        addr string,
        plugin PluginIntfs,
        timeout time.Duration,
        qps uint32,
        duration time.Duration,
        concurrency uint32,
        keepalive bool,
        resultChan chan *CallResult) (UnicornIntfs, error) {

    log.Logger.Info("Begin New Unicorn")

    //参数校验
    if (qps != 0 && concurrency != 0) ||
          (qps == 0 && concurrency == 0) {
        //qps和concurrency不能同时为0，或者同时不为0
        return nil, errors.New("qps and concurrency can't be 0 all. or is 0 all!")
    }
    if plugin == nil {
        return nil, errors.New("Nil plugin")
    }
    if timeout == 0 {
        return nil, errors.New("Nil timeout")
    }
    if duration == 0 {
        return nil, errors.New("Nil duration")
    }
    if resultChan == nil {
        return nil, errors.New("Nil resultChan")
    }
    if addr == "" {
        return nil, errors.New("Nil address")
    }

    //如果并发量为空，则需要自行计算并发量，这个计算方法比较晦涩，如果自己指定并发量比较容易理解，但不够科学
    //concurrency ≈ 规定的最大响应超时时间 / 规定的发送间隔时间
    var c uint32
    if qps != 0 {
        var conc int64 = int64(timeout) / int64(1e9/ qps) + 1
        if conc > math.MaxInt32 {
            conc = math.MaxInt32
        }
        c = uint32(conc)
        log.Logger.Info("Concurrency auto calculate： " + fmt.Sprintf("%d" ,c))
    } else if concurrency != 0 {
        c = concurrency
    } else {
        log.Logger.Fatal("qps and concurrency can't be 0 all. or is 0 all!")
    }

    //设定节流阀(利用断续器，实现的循环定时事件，用于控制请求发出的频率)
    var throttle <-chan time.Time
    if qps > 0 {
        interval := time.Duration(1e9 / qps) //发送每个请求的间隔
        throttle = time.Tick(interval)
        log.Logger.Info("The interval of per request is " + fmt.Sprintf("%d" ,interval) + " Nanosecond")
    }

    //实例化goroutine池
    pool, err := wp.NewWorkerPool(c)
    if err != nil {
        return nil, err
    }

    //创建instance
    unc := &Unicorn{
        serverAdd  : addr,
        plugin     : plugin,
        timeout    : timeout,
        qps        : qps,
        duration   : duration,
        concurrency: c,
        sigChan    : make(chan byte, 1),
        stopFlag   : false,
        status     : ORIGINAL,
        resultChan : resultChan,
        finalCnt   : 0,
        pool       : pool,
        throttle   : throttle,
        keepalive  : keepalive,
    }

    return unc, nil
}

/******************** Unicorn实现Unicorn接口 *******************/
//启动
func (unc *Unicorn)Start() *sync.WaitGroup{
    log.Logger.Info("Unicorn Start...")

    //停止定时器，当探测持续到了指定时间，停止unicorn
    time.AfterFunc(unc.duration, func(){
        log.Logger.Info("Time's up. Sending Stop signal...")
        unc.sigChan <- 1
    })

    //启动状态
    unc.status = STARTED

    var wg sync.WaitGroup
    wg.Add(1)

    //因为Start属于主goroutine，不应该有被阻塞住的可能性。主goroutine是应该最外层起到整体管理的作用。
    //而doRequest是存在被阻塞住的可能性的，即协程池的Take操作，所以启动独立的goroutine
    go unc.doRequest(&wg)

    return &wg
}

//停止，返回值表示停止时已完成请求数和否成功停止
func (unc *Unicorn) Stop() (uint64, bool){
    if unc.sigChan == nil {
        return 0, false
    }
    if unc.status != STARTED {
        return 0, false
    }
    unc.status = STOPPED
    unc.sigChan <- 1      //放入标记

    //call_count := <-unc.finalCnt
    //log.Logger.Info(fmt.Sprintf("Stop ended. (callCount=%d)", call_count))
    return unc.finalCnt, true
}

//获得unicorn当前状态
func (unc *Unicorn) Status() UncStatus {
    return unc.status
}

/*
 * 发送请求的总控制逻辑，放置在独立的goroutine中执行
 */
func (unc* Unicorn) doRequest(wg *sync.WaitGroup) {
    log.Logger.Info("doRequest ...")

    //无限循环，保持足够多的worker，即维持concurrency数量的worker
    for {
        if unc.stopFlag {
            break
        }
        //产生worker，异步发送请求（此处是可能被阻塞--协程池的Take操作）
        unc.createWorker()
    }
    //等待所有worker归还票
    for {
        if unc.pool.Remainder() == unc.concurrency {
            break
        }
        time.Sleep(10*time.Millisecond)
    }

    unc.status = STOPPED
    log.Logger.Info(fmt.Sprintf("doRequest ended. (callCount=%d)", unc.finalCnt))
    wg.Done()
}

/*
 * 结果码转成字符串
 */
func ConvertCodePlain(code ResultCode) string {
    var code_plain string
    switch code {
    case RESULT_CODE_SUCCESS:
        code_plain = "Success"
    case RESULT_CODE_WARING_TIMEOUT:
        code_plain = "Call Timeout Warning"
    case RESULT_CODE_ERROR_CALL:
        code_plain = "Call Error"
    case RESULT_CODE_ERROR_RESPONSE:
        code_plain = "Response Error"
    case RESULT_CODE_ERROR_CALEE:
        code_plain = "Callee Error"
    case RESULT_CODE_FATAL_CALL:
        code_plain = "Call Fatal Error"
    default:
        code_plain = "Unknown result code"
    }
    return code_plain
}