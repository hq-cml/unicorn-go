package unicorn

/*
 * worker逻辑
 */
import (
    "time"
    "fmt"
    "errors"
    "bytes"
    "github.com/hq-cml/unicorn-go/log"
    "net"
    "bufio"
    "io"
)

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

//归还worker_pool票
func (unc *Unicorn) returnTicket() {
    unc.pool.Return() //还票
}

//生成goroutine，发送请求，利用worker_pool，控制goroutine总量
func (unc *Unicorn) createWorker() {
    //Take和Return时机很重要，必须是父goroutine申请，子goroutine归还！否则无法起到控制goroutine的作用
    unc.pool.Take()
    //worker启动：子goroutine
    go func() {
        //注册defer：错误处理
        defer unc.handleError()

        //注册defer：归还票，多个defer会逆序执行
        defer unc.returnTicket()

        //如果程序停止，则退出
        if unc.stopFlag {
            return
        }

        //和服务端建立连接
        conn, err := net.DialTimeout("tcp", unc.serverAdd, unc.timeout)
        if err != nil {
            return
        }

        //注册defer：关闭连接
        defer func(){ conn.Close() }()

        //开启探测
        for {
            //检查停止信号（default--非阻塞式检查）
            select {
            case <-unc.sigChan:  //停止信号
                //fmt.Println("Recv stop sig")
                unc.handleStopSign()
            default:
            }

            //如果节流阀非空（说明初始设置了qps），则利用节流阀进行频率控制（阻塞式等待）
            if unc.throttle != nil {
                select {
                case <-unc.throttle: //throttle用来控制发送频率，其实本身是空转一次不作实质事情
                }
            }
            //如果程序停止，则退出
            if unc.stopFlag {
                return
            }

            //构造请求
            id := time.Now().UnixNano() //用纳秒就能保证唯一性了吗？
            raw_request := unc.plugin.GenRequest(id)

            //启动一个异步定时器，check超时
            var timeout_flag = false
            timer := time.AfterFunc(unc.timeout, func(){
                timeout_flag = true
            })

            //同步交互：发送请求+接收响应
            start := time.Now().Nanosecond()
            data, err := unc.interact(&raw_request, conn)
            end := time.Now().Nanosecond()

            //上面是一个同步的过程，所以到了此处，可能是已经超时了
            //所以检测超时标志，只有未超时，才有必要继续
            var result *CallResult
            if err != nil {
                timer.Stop() //!!立刻停止异步定时器
                result = &CallResult{
                    Id     : id,
                    //Req    : raw_request,
                    Code   : RESULT_CODE_ERROR_CALL,
                    Msg    : err.Error(),
                    Elapse : time.Duration(end - start),
                }
            }else if timeout_flag {
                result = &CallResult{
                    Id     : raw_request.Id,
                    //Req    : raw_request,
                    Code   : RESULT_CODE_WARING_TIMEOUT,
                    Msg    : fmt.Sprintf("Timeout! (expected: < %v)", unc.timeout),
                    Elapse : time.Duration(end - start),
                }
            } else {
                timer.Stop() //!!立刻停止异步定时器
                result = unc.plugin.CheckResponse(raw_request, data)
                result.Elapse = time.Duration(end - start)
            }

            unc.saveResult(result) //结果存入通道

            //如果不是长连接模式，则退出
            if !unc.keepalive {
                break
            }
            //fmt.Println("Go on")
        }
    }()
}

//实际的交互逻辑
func (unc *Unicorn)interact(raw_request *RawRequest, conn net.Conn) ([]byte, error){
    //总请求计数+1
    unc.AllCnt ++

    //发送请求
    n, err := sendRequest(conn, raw_request.Req)
    if err != nil {
        return nil, err
    }
    _ = n

    data := make([]byte, 0)
    Loop:
    for {
        buf, n, err := recvResponse(conn)
        if err != nil && err != io.EOF {
            return nil, err
        } else if err == io.EOF {
            //服务端关闭连接，通常，服务端不会主动关闭连接
            log.Logger.Info("Server close connection!")
            return nil, err
        } else {
            data = append(data, buf[0:n]...)
            switch unc.plugin.CheckFull(raw_request, data) {
            case SER_OK:
                break Loop
            case SER_NEEDMORE:
                continue Loop
            default:
                err = errors.New("Sth Wrong!")
                break Loop
            }
        }
    }
    return data, nil
}

//请求发送
func sendRequest(conn net.Conn, content []byte) (int, error) {
    //利用带缓冲的Writer
    writer := bufio.NewWriter(conn)
    n, err := writer.Write(content) //Write内部可以保证content全部内容写入到了缓冲

    if err == nil {
        err = writer.Flush() //将缓冲刷向网络
    }
    return n, err
}

//接收请求
func recvResponse(conn net.Conn) ([]byte, int, error) {
    buf := make([]byte, 1024)

    n, err := conn.Read(buf)
    if err != nil {
        return nil, 0, err
    }

    return buf, n, nil
}

//保存结果:将结果存入通道
func (unc *Unicorn) saveResult(result *CallResult) bool {
    if unc.status == STOPPED && unc.stopFlag {
        unc.IgnoreCnt++
        log.Logger.Info("Ignore result :" + fmt.Sprintf("Id=%d, Code=%d, Msg=%s, Elaspe=%v", result.Id, result.Code, result.Msg, result.Elapse))
        return false
    }
    unc.resultChan <- result
    return true
}

//处理停止“信号”
func (unc * Unicorn) handleStopSign() {
    //信号标记变为1
    unc.stopFlag = true
    log.Logger.Info("handleStopSign. Closing result chan...")

    //关闭结果存储通道 -- 这个地方关闭不合理，应该放在外部统一关闭
    //close(unc.resultChan)
}
