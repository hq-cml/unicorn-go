package plugin

import (
    "testing"
    "runtime"
    "github.com/hq-cml/unicorn-go/unicorn"
    "time"
    "fmt"
    "net"
    "sync"
    "encoding/json"
    "errors"
    "strconv"
    "bytes"
)

var printDetail = true

//测试Start方法
func TestStart(t *testing.T) {
    runtime.GOMAXPROCS(runtime.NumCPU())

    //初始化Server
    server := NewTcpServer()
    defer server.Close() //注册关闭
    addr := "127.0.0.1:9527"
    t.Logf("Startup Tcp Server(%s)..\n", addr)
    err := server.Listen(addr)
    if err != nil {
        t.Fatalf("TCP Server startup failing! (addr=%s)!\n", addr)
        t.FailNow() //结束！
    }

    //初始化Plugin
    plugin_tep := NewTcpEquationPlugin(addr)

    //初始化Unicorn
    result_chan := make(chan *unicorn.CallResult, 50)
    timeout := 10*time.Millisecond
    qps := uint32(1000)
    duration := 10 * time.Second
    t.Logf("Initialize Unicorn (timeout=%v, qps=%d, duration=%v)...", timeout, qps, duration)

    unc, err := unicorn.NewUnicorn(plugin_tep, timeout, qps, duration, 0, result_chan)
    if err != nil {
        t.Fatalf("Unicorn initialization failing: %s.\n",  err)
        t.FailNow()
    }

    //开始干活儿! Start可以立刻返回的，进去看就知道~
    t.Log("Start Unicorn...")
    unc.Start()

    //主流程在外面做一些总体控制工作，比如，循环阻塞接收结果~
    count_map := make(map[unicorn.ResultCode]int) //将结果按Code分类收集
    for ret := range result_chan {
        count_map[ret.Code] ++
        if printDetail && ret.Code != unicorn.RESULT_CODE_SUCCESS{
            time := fmt.Sprintf(time.Now().Format("15:04:05"))
            t.Logf("[%s] Result: Id=%d, Code=%d, Msg=%s, Elapse=%v.\n", time, ret.Id, ret.Code, ret.Msg, ret.Elapse)
        }
    }

    //打印汇总结果
    var total int
    t.Log("Code Count:")
    for k, v := range count_map {
        code_plain := unicorn.ConvertCodePlain(k)
        t.Logf("  Code plain: %s (%d), Count: %d.\n", code_plain, k, v)
        total += v
    }

    //打印最终结果
    t.Logf("Total load: %d.\n", total)
    success_cnt := count_map[unicorn.RESULT_CODE_SUCCESS]
    tps := float64(success_cnt) / float64(duration/time.Second)
    t.Logf("Qps: %d; Tps(Treatments per second): %f.\n", qps, tps)
}

//测试手动停止
func TestStop(t *testing.T) {
    runtime.GOMAXPROCS(runtime.NumCPU())

    //初始化Server
    server := NewTcpServer()
    defer server.Close() //注册关闭
    addr := "127.0.0.1:9527"
    t.Logf("Startup Tcp Server(%s)..\n", addr)
    err := server.Listen(addr)
    if err != nil {
        t.Fatalf("TCP Server startup failing! (addr=%s)!\n", addr)
        t.FailNow() //结束！
    }

    //初始化Plugin
    plugin_tep := NewTcpEquationPlugin(addr)

    //初始化Unicorn
    result_chan := make(chan *unicorn.CallResult, 50)
    timeout := 3*time.Millisecond
    qps := uint32(10)
    duration := 10 * time.Second
    t.Logf("Initialize Unicorn (timeout=%v, qps=%d, duration=%v)...", timeout, qps, duration)
    unc, err := unicorn.NewUnicorn(plugin_tep, timeout, qps, duration, 0, result_chan)
    if err != nil {
        t.Fatalf("Unicorn initialization failing: %s.\n",  err)
        t.FailNow()
    }

    //开始干活儿! Start可以立刻返回的，进去看就知道~
    t.Log("Start Unicorn...")
    unc.Start()

    //主流程在外面等待着结果接收，循环阻塞接收结果~
    //利用count，在4次之后，手动显式停止Unicorn
    count := 0
    count_map := make(map[unicorn.ResultCode]int) //将结果按Code分类收集
    for ret := range result_chan {
        count_map[ret.Code] ++
        if printDetail {
            t.Logf("Result: Id=%d, Code=%d, Msg=%s, Elapse=%v.\n", ret.Id, ret.Code, ret.Msg, ret.Elapse)
        }
        count ++
        if count > 3 {
            unc.Stop() //显式地停止
        }
    }

    //打印汇总结果
    var total int
    t.Log("Code Count:")
    for k, v := range count_map {
        code_plain := unicorn.ConvertCodePlain(k)
        t.Logf("  Code plain: %s (%d), Count: %d.\n", code_plain, k, v)
        total += v
    }

    //打印最终结果
    t.Logf("Total load: %d.\n", total)
    success_cnt := count_map[unicorn.RESULT_CODE_SUCCESS]
    tps := float64(success_cnt) / float64(duration/time.Second)
    t.Logf("Qps: %d; Tps(Treatments per second): %f.\n", qps, tps)
}

/**************************配套服务端的逻辑********************/
type TcpServer struct {
    listener net.Listener
    active   bool
    lock     *sync.Mutex
}

func (self *TcpServer) init(addr string) error {
    self.lock.Lock()
    defer self.lock.Unlock()
    if self.active {
        return nil
    }
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        return err
    }
    self.listener = ln
    self.active = true
    return nil
}

func (self *TcpServer) Listen(addr string) error {
    err := self.init(addr)
    if err != nil {
        return err
    }
    go func(active *bool) {
        for {
            conn, err := self.listener.Accept()
            if err != nil {
                fmt.Errorf("Server: Request Acception Error: %s\n", err)
                continue
            }
            go reqHandler(conn)
            runtime.Gosched()
        }
    }(&self.active)
    return nil
}

func NewTcpServer() *TcpServer {
    return &TcpServer{lock: new(sync.Mutex)}
}

func (self *TcpServer) Close() bool {
    self.lock.Lock()
    defer self.lock.Unlock()
    if self.active {
        self.listener.Close()
        self.active = false
        return true
    } else {
        return false
    }
}

func reqHandler(conn net.Conn) {
    var errMsg string
    var sresp ServerEquationResp
    req, err := read(conn, DELIM)
    if err != nil {
        errMsg = fmt.Sprintf("Server: Req Read Error: %s", err)
    } else {
        var sreq ServerEquationReq
        err := json.Unmarshal(req, &sreq)
        if err != nil {
            errMsg = fmt.Sprintf("Server: Req Unmarshal Error: %s", err)
        } else {
            sresp.Id = sreq.Id
            sresp.Result = op(sreq.Operands, sreq.Operator)
            sresp.Formula = genFormula(sreq.Operands, sreq.Operator, sresp.Result, true)
        }
    }
    if errMsg != "" {
        sresp.Err = errors.New(errMsg)
    }
    bytes, err := json.Marshal(sresp)
    if err != nil {
        fmt.Errorf("Server: Resp Marshal Error: %s", err)
    }
    _, err = write(conn, bytes, DELIM)
    if err != nil {
        fmt.Errorf("Server: Resp Write error: %s", err)
    }
}

func genFormula(operands []int, operator string, result int, equal bool) string {
    var buff bytes.Buffer
    n := len(operands)
    for i := 0; i < n; i++ {
        if i > 0 {
            buff.WriteString(" ")
            buff.WriteString(operator)
            buff.WriteString(" ")
        }

        buff.WriteString(strconv.Itoa(operands[i]))
    }
    if equal {
        buff.WriteString(" = ")
    } else {
        buff.WriteString(" != ")
    }
    buff.WriteString(strconv.Itoa(result))
    return buff.String()
}