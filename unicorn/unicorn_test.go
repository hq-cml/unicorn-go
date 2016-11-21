package unicorn

import (
    "testing"
    "runtime"
    "github.com/hq-cml/unicorn-go/unicorn"
    "github.com/hq-cml/unicorn-go/plugin"
    "time"
    "fmt"
)

var printDetail = true

func TestStart(t *testing.T) {
    runtime.GOMAXPROCS(runtime.NumCPU())

    //初始化Server
    server := plugin.NewTcpServer()
    defer server.Close() //注册关闭
    addr := "127.0.0.1:9527"
    t.Logf("Startup Tcp Server(%s)..\n", addr)
    err := server.Listen(addr)
    if err != nil {
        t.Fatalf("TCP Server startup failing! (addr=%s)!\n", addr)
        t.FailNow() //结束！
    }

    //初始化Plugin
    plugin_tep := plugin.NewTcpEquationPlugin(addr)

    //初始化Unicorn
    result_chan := make(chan *unicorn.CallResult, 50)
    timeout := 10*time.Millisecond
    qps := uint32(1000)
    duration := 10 * time.Second
    t.Logf("Initialize Unicorn (timeout=%v, qps=%d, duration=%v)...", timeout, qps, duration)
    unc, err := NewUnicorn(plugin_tep, timeout, qps, duration, result_chan)
    if err != nil {
        t.Fatalf("Unicorn initialization failing: %s.\n",  err)
        t.FailNow()
    }

    //开始干活儿! Start可以立刻返回的，进去看就知道~
    t.Log("Start Unicorn...")
    unc.Start()

    //主流程在外面等待着结果接收，循环阻塞接收结果~
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
    server := plugin.NewTcpServer()
    defer server.Close() //注册关闭
    addr := "127.0.0.1:9527"
    t.Logf("Startup Tcp Server(%s)..\n", addr)
    err := server.Listen(addr)
    if err != nil {
        t.Fatalf("TCP Server startup failing! (addr=%s)!\n", addr)
        t.FailNow() //结束！
    }

    //初始化Plugin
    plugin_tep := plugin.NewTcpEquationPlugin(addr)

    //初始化Unicorn
    result_chan := make(chan *unicorn.CallResult, 50)
    timeout := 3*time.Millisecond
    qps := uint32(10)
    duration := 10 * time.Second
    t.Logf("Initialize Unicorn (timeout=%v, qps=%d, duration=%v)...", timeout, qps, duration)
    unc, err := NewUnicorn(plugin_tep, timeout, qps, duration, result_chan)
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