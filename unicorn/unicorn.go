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
        plugin     : plugin,
        timeout    : timeout,
        qps        : qps,
        duration   : duration,
        concurrency   : c,
        sigChan    : make(chan byte, 1),
        stopFlag : false,
        status     : ORIGINAL,
        resultChan : resultChan,
        //finalCnt : make(chan uint64, 2),
        finalCnt   : 0,
        pool: pool,
        throttle: throttle,
        keepalive: keepalive,
    }

    return unc, nil
}

/******************** Unicorn实现Unicorn接口 *******************/
//启动
func (unc *Unicorn)Start() {
    log.Logger.Info("Unicorn Start...")

    //停止定时器，当探测持续到了指定时间，能够停止unicorn
    //实际测试，这个地方是否启动一个goroutine，效果是一样的
    //go func() { // ??为何要单独一个goroutinue
    time.AfterFunc(unc.duration, func(){
        log.Logger.Info("Time's up. Stoping Unicorn...")
        unc.sigChan <- 1
    })
    //}()

    //启动状态
    unc.status = STARTED

    //放在独立的goroutine中，使得请求的发送工作异步化。因为Start是主goroutine，不应该有被阻塞住的可能性。
    //主goroutine是应该最外层起到整体管理的作用，doRequest是存在被阻塞住的可能性的，即协程池的Take操作
    go func() {
        log.Logger.Info("doRequest ...")
        //这是一个同步的过程(因为存在协程池的Tack操作）
        //unc.doRequest()

        //无限循环，保持足够多的worker，保持concurrency
        for {
            //异步发送请求（此处是有可能被阻塞住的--协程池的Take操作）
            //unc.asyncSendRequest()
            unc.createWorker()
        }

        //TODO 等待还票？

        //接收最终个数
        //call_count := <-unc.finalCnt
        unc.status = STOPPED

        log.Logger.Info(fmt.Sprintf("Start go func ended. (callCount=%d)", unc.finalCnt))
    }()
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

/******************** 其他核心函数 *******************/
//处理停止“信号”
func (unc * Unicorn) handleStopSign() {
    //信号标记变为1
    unc.stopFlag = true
    log.Logger.Info("handleStopSign. Closing result chan...")
    //关闭结果存储通道
    close(unc.resultChan)

    //unc.finalCnt <- call_cnt
    ////为什么需要两次写入通道呢
    ////因为Start方法和Stop方法，均存在从finalCnt接收的情况，所以如果两个同时发生，会造成其中一个阻塞
    ////所以，索性写入两次，保证Start和Stop均不会阻塞！
    //unc.finalCnt <- call_cnt
}

/*
 * 发送请求的总控制逻辑
 * 通过节流阀throttle控制发送请求的频度
 * 请求过程中不断检测stopSign，如果检测到，则将最终结果传入finalCnt
 */
func (unc* Unicorn) doRequest() {
    Loop:
    //一个无限循环，产生足够多的worker，保持concurrency
    for {
        //带default的select分支，不会阻塞，放在这里为了能够及时收到Stop信号，但感觉没太大必要
        //select {
        //case <- unc.sigChan:
        //    unc.handleStopSign(call_cnt)
        //    break Loop
        //default:
        //}

        //异步发送请求（此处是有可能被阻塞住的--协程池的Take操作）
        unc.createWorker()

        //因为新增了长连接模式，所以asyncSendRequest可能长时间不返回了（因为woker们会保持连接持续发送请求），所以下面的sigChan信号接收位置需要调整到worker内部
        //阻塞等待节流阀throttle信号
        //select {
        //case <-unc.throttle:     //throttle用来控制发送频率，其实本身是空转一次不作实质事情，进入下次循环，发送请求
        //case <-unc.sigChan:  //停止信号
        //    unc.handleStopSign(call_cnt)
        //    break Loop
        //}

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
        log.Logger.Info(msg)

        //填充结果
        result := &CallResult {
            Id     : -1,
            Code   : RESULT_CODE_FATAL_CALL,
            Msg    : msg,
        }
        //结果存入通道
        unc.saveResult(result)
    }
}

//生成goroutine，发送请求，利用worker_pool，控制goroutine总量
func (unc *Unicorn) createWorker()() {
    //Take和Return时机很重要，必须是主goroutine申请，子goroutine归还！
    //这个时机如果不正确，就无法起到控制goroutine的作用
    unc.pool.Take() //主goroutine申请派生

    //子goroutine
    go func() {
        //注册错误处理，//TODO 错误处理里面需要归还票吗？
        defer unc.handleError()

        //检查停止信号
        select {
        case <-unc.sigChan:  //停止信号
            unc.handleStopSign()
        default:
        }

        //如果节流阀非空（说明设置了qps），则利用节流阀进行频率控制
        if unc.throttle != nil {
            select {
            case <-unc.throttle:     //throttle用来控制发送频率，其实本身是空转一次不作实质事情，进入下次循环，发送请求
            }
        }


        //如果程序停止，则退出
        if unc.stopFlag {
            return
        }

        //构造请求
        raw_request := unc.plugin.GenRequest()

        //启动一个异步定时器
        var timeout_flag = false
        timer := time.AfterFunc(unc.timeout, func(){
            timeout_flag = true
            result := &CallResult{
                Id     : raw_request.Id,
                Req    : raw_request,
                Code   : RESULT_CODE_WARING_TIMEOUT,
                Msg    : fmt.Sprintf("Timeout! (expected: < %v)", unc.timeout),
            }
            unc.saveResult(result) //结果存入通道
        })

        //同步交互,调用plugin的Call方法获得response
        raw_response := unc.interact(&raw_request)

        //上面是一个同步的过程，所以到了此处，可能是已经超时了
        //所以检测超时标志，只有未超时，才有必要继续
        if !timeout_flag {
            timer.Stop() //!!立刻停止异步定时器，防止异步的方法执行，写入了一个超时结果
            var result *CallResult
            if raw_response.Err != nil {
                result = &CallResult{
                    Id     : raw_response.Id,
                    Req    : raw_request,
                    Code   : RESULT_CODE_ERROR_CALL,
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

//实际的交互逻辑，调用了plugin.call函数
func (unc *Unicorn)interact(raw_request *RawRequest) *RawResponse{
    var raw_response *RawResponse
    if raw_request == nil {
        raw_response = &RawResponse{
            Id: -1,
            Err: errors.New("Invalid raw request."),
        }
    } else {
        start := time.Now().Nanosecond()
        resp, err := unc.plugin.Call(raw_request.Req, unc.timeout)
        end := time.Now().Nanosecond()
        if err != nil {
            errMsg := fmt.Sprintf("Sync call Error: %s", err)
            raw_response = &RawResponse{
                Id: raw_request.Id,
                Err: errors.New(errMsg),
                Elapse: time.Duration(end - start),
            }
        } else {
            raw_response = &RawResponse{
                Id : raw_request.Id,
                Resp: resp,
                Elapse: time.Duration(end - start),
            }
        }
    }
    unc.finalCnt ++ //总调用计数+1
    return raw_response
}

//保存结果:将结果存入通道
func (unc *Unicorn) saveResult(result *CallResult) bool {
    if unc.status == STARTED && unc.stopFlag == false {
        unc.resultChan <- result
        return true
    }
    log.Logger.Info("Ignore result :" + fmt.Sprintf("Id=%d, Code=%d, Msg=%s, Elaspe=%v", result.Id, result.Code, result.Msg, result.Elapse))
    return false
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