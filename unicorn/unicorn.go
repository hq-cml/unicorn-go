package unicorn

import (
    "github.com/hq-cml/unicorn-go/log"
    wp "github.com/hq-cml/unicorn-go/worker-pool"
    "math"
    "fmt"
    "time"
    "errors"
    "bytes"
)

/*
 * 惯例New函数，实例化Unicorn，返回的是UnicornIntfs的实现
 */
func NewUnicorn(
        plugin PluginIntfs,
        timeout time.Duration,
        qps uint32,
        duration time.Duration,
        concurrency uint32,
        resultChan chan *CallResult) (UnicornIntfs, error) {

    log.Logger.Info("Begin New Unicorn")

    //参数校验
    if plugin == nil {
        return nil, errors.New("Nil plugin")
    }
    if timeout == 0 {
        return nil, errors.New("Nil timeout")
    }
    if qps == 0 {
        return nil, errors.New("Nil qps")
    }
    if duration == 0 {
        return nil, errors.New("Nil duration")
    }
    if resultChan == nil {
        return nil, errors.New("Nil resultChan")
    }

    //如果并发量为空，则需要自行计算并发量，这个计算方法比较晦涩，如果自己指定并发量比较容易理解，但不够科学
    //concurrency ≈ 规定的最大响应超时时间 / 规定的发送间隔时间
    c := concurrency
    if c == 0 {
        var conc int64 = int64(timeout) / int64(1e9/ qps) + 1
        if conc > math.MaxInt32 {
            conc = math.MaxInt32
        }
        c = uint32(conc)
        log.Logger.Info("Concurrency auto calculate： " + fmt.Sprintf("%d" ,c))
    }

    //实例化线程池
    pool, err := wp.NewWorkerPool(c)
    if err != nil {
        return nil, err
    }

    //创建instance
    unc := &Unicorn{
        plugin     : plugin,
        timeout    : timeout,
        qps        : qps,
        duration   : duration,
        concurrency   : c,
        sigChan    : make(chan byte, 1),
        cancelSign : 0,
        status     : unicorn.ORIGINAL,
        resultChan : resultChan,
        finalCnt   : make(chan uint64, 2),
        pool: pool,
    }

    //初始化
    if err := unc.init(); err != nil {
        return nil, err
    }
    return unc, nil
}

//*Unicorn实现Unicorn接口
//启动
func (unc *Unicorn)Start() {
    unicorn.Logger.Info("Unicorn Start...")

    //设定节流阀(利用断续器，实现的循环定时事件)
    var throttle <-chan time.Time
    if unc.qps > 0 {
        interval := time.Duration(1e9 / unc.qps) //发送每个请求的间隔
        throttle = time.Tick(interval)
        unicorn.Logger.Info("The interval of per request is " + fmt.Sprintf("%d" ,interval) + " Nanosecond")
    }

    //停止定时器，当探测持续到了指定时间，能够停止unicorn
    //实际测试，这个地方是否启动一个goroutine，效果是一样的
    //go func() { // ??为何要单独一个goroutinue
    time.AfterFunc(unc.duration, func(){
        unicorn.Logger.Info("Time's up. Stoping Unicorn...")
        unc.sigChan <- 0
    })
    //}()

    // 初始化完结信号通道
    //unc.finalCnt = make(chan uint64, 1) //放在NewUnicorn里面可以吗

    //启动状态
    unc.status = unicorn.STARTED

    //这个地方为何要用goroutine呢
    //因为Start是主流程，不应该有被阻塞住的可能性，需要能够被执行完毕进而继续外层和Start平行的逻辑
    //仔细看beginRequest是存在被阻塞住的可能性的，即协程池的Take操作，所以此处应该启动一个单独goroutine
    go func() {
        unicorn.Logger.Info("genRequest ...")
        //这是一个同步的过程
        unc.beginRequest(throttle)

        //接收最终个数
        call_count := <-unc.finalCnt
        unc.status = unicorn.STOPPED

        unicorn.Logger.Info(fmt.Sprintf("Start go func ended. (callCount=%d)", call_count))
    }()
}

//停止，返回值表示停止时已完成请求数和否成功停止
func (unc *Unicorn) Stop() (uint64, bool){
    if unc.sigChan == nil {
        return 0, false
    }
    if unc.status != unicorn.STARTED {
        return 0, false
    }
    unc.status = unicorn.STOPPED
    unc.sigChan <- 1
    //time.Sleep(1) //模拟让Start方法先接收
    call_count := <-unc.finalCnt
    unicorn.Logger.Info(fmt.Sprintf("Stop ended. (callCount=%d)", call_count))
    return call_count, true
    return 0, true
}

//获得unicorn当前状态
func (unc *Unicorn) Status() unicorn.UncStatus {
    return unc.status
}

//处理停止“信号”
func (unc * Unicorn) handleStopSign(call_cnt uint64) {
    //信号标记变为1
    unc.cancelSign = 1
    unicorn.Logger.Info("handleStopSign. Closing result chan...")
    //关闭结果存储通道
    close(unc.resultChan)
    unc.finalCnt <- call_cnt
    //为什么需要两次写入通道呢
    //因为Start方法和Stop方法，均存在从finalCnt接收的情况，所以如果两个同时发生，会造成其中一个阻塞
    //所以，索性写入两次，保证Start和Stop均不会阻塞！
    unc.finalCnt <- call_cnt
}

//发送请求的主控制逻辑
//通过节流阀throttle控制发送请求的强度
//请求过程中不断检测stopSign，如果检测到，则将最终结果传入finalCnt
func (unc* Unicorn) beginRequest(throttle <-chan time.Time) {
    call_cnt := uint64(0)

    Loop:
    //一个无限循环，只要满足条件，就发送请求
    for ;;call_cnt++ {
        //带default的select分支，是不会出现阻塞的
        select {
        case <- unc.sigChan:
            unc.handleStopSign(call_cnt)
            break Loop
        default:
        }

        //异步发送请求（此处是有可能被阻塞住的--协程池的Take操作）
        unc.asyncSendRequest()

        //阻塞等待节流阀throttle信号
        if unc.qps > 0 {
            select {
            case <-throttle: //空转一次，进入下次循环，发送请求
            case <-unc.sigChan:
                unc.handleStopSign(call_cnt)
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
        //recover的返回值，静态类型是interface，动态类型未知，因此需要类型断言
        err, ok := interface{}(p).(error)
        if ok { //断言成功
            buff.WriteString("error :" + err.Error())
        } else {
            buff.WriteString("clue :" + fmt.Sprintf("%v", p))
        }
        buff.WriteString(")")
        msg := buff.String()
        unicorn.Logger.Info(msg)

        //填充结果
        result := &unicorn.CallResult {
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
func (unc *Unicorn) asyncSendRequest() {
    //Take和Return时机很重要，必须是主goroutine申请，子goroutine归还！
    //这个时机如果不正确，就无法起到控制goroutine的作用
    unc.pool.Take() //主goroutine申请派生

    //子goroutine
    go func() {
        //注册错误处理
        defer unc.handleError()

        //构造请求
        raw_request := unc.plugin.GenRequest()

        //启动一个异步定时器
        var timeout_flag = false
        timer := time.AfterFunc(unc.timeout, func(){
            timeout_flag = true
            result := &unicorn.CallResult{
                Id     : raw_request.Id,
                Req    : raw_request,
                Code   : unicorn.RESULT_CODE_WARING_TIMEOUT,
                Msg    : fmt.Sprintf("Timeout! (expected: < %v)", unc.timeout),
            }
            unc.saveResult(result) //结果存入通道
        })

        //同步交互,调用plugin的Call方法获得response
        raw_response := unc.interact(&raw_request)

        //上面是一个同步的过程，所以到了此处，可能是已经超时了
        //所以检测超时标志，只有未超时，才有必要继续
        if !timeout_flag {
            timer.Stop() //立刻停止异步定时器，防止异步的方法执行，写入了一个超时结果
            var result *unicorn.CallResult
            if raw_response.Err != nil {
                result = &unicorn.CallResult{
                    Id     : raw_response.Id,
                    Req    : raw_request,
                    Code   : unicorn.RESULT_CODE_ERROR_CALL,
                    Msg    : raw_response.Err.Error(),
                    Elapse : raw_response.Elapse,
                }
            }else {
                result = unc.plugin.CheckResponse(raw_request, *raw_response)
                result.Elapse = raw_response.Elapse
            }
            unc.saveResult(result) //结果存入通道
        }
        unc.pool.Return() //子goroutine归还
    }()
}

//TODO 这个函数写入程序中去。。。
func (unc *Unicorn)interact(raw_request *unicorn.RawRequest) *unicorn.RawResponse{
    var raw_response *unicorn.RawResponse
    if raw_request == nil {
        raw_response = &unicorn.RawResponse{
            Id: -1,
            Err: errors.New("Invalid raw request."),
        }
    } else {
        start := time.Now().Nanosecond()
        resp, err := unc.plugin.Call(raw_request.Req, unc.timeout)
        end := time.Now().Nanosecond()
        if err != nil {
            errMsg := fmt.Sprintf("Sync call Error: %s", err)
            raw_response = &unicorn.RawResponse{
                Id: raw_request.Id,
                Err: errors.New(errMsg),
                Elapse: time.Duration(end - start),
            }
        } else {
            raw_response = &unicorn.RawResponse{
                Id : raw_request.Id,
                Resp: resp,
                Elapse: time.Duration(end - start),
            }
        }
    }
    return raw_response
}

//保存结果:将结果存入通道
func (unc *Unicorn) saveResult(result *unicorn.CallResult) bool {
    if unc.status == unicorn.STARTED && unc.cancelSign == 0 {
        unc.resultChan <- result
        return true
    }
    unicorn.Logger.Info("Ignore result :" + fmt.Sprintf("Id=%d, Code=%d, Msg=%s, Elaspe=%v", result.Id, result.Code, result.Msg, result.Elapse))
    return false
}

/*
//注释的方案也是一种可行方案，但是有一个严重的问题就是会启动孙goroutine来实施交互
//子协程则负责访问超时的判断。这样的结果就是导致最多会有worker_pool容量两倍的goroutine
//被创建出来，所以不推荐这个方案
func (unc *Unicorn) asyncSendRequest() {
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
*/

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