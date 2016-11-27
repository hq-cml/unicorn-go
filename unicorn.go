package main

import (
    "runtime"
    "flag"
    "fmt"
    "github.com/hq-cml/unicorn-go/log"
    "github.com/hq-cml/unicorn-go/unicorn"
    "github.com/hq-cml/unicorn-go/plugin"
    "time"
)

var ip *string = flag.String("h", "127.0.0.1", "ip")
var port *string = flag.String("p", "9527", "port")
var c *int = flag.Int("c", 1, "concurrency")
var q *int = flag.Int("q", 10, "qps")
var t *int = flag.Int("q", 50, "timeout")
var D *int = flag.Int("D", 5, "port")
var k *bool = flag.Bool("k", true, "keep alive")
var H *bool = flag.Bool("H", false, "help")
var v *bool = flag.Bool("v", false, "verbose")

func showUseage() {
    fmt.Println()
    fmt.Println("Usage: unicorn [-h <ip>] [-p <port>] [-c <concurrency>] [-D duration]> [-k <boolean>]");
    fmt.Println("Note: !!!!- The argu 'c' and 'q' can't be set at the same time -!!!!");
    fmt.Println()
    fmt.Println(" -h <hostname>      server hostname (default 127.0.0.1)");
    fmt.Println(" -p <port>          server port (default 9527)");
    fmt.Println(" -c <concurrency>   number of parallel connections (default 1)");
    fmt.Println(" -q <qps>           qps-- the frequence you wanted for requests (default 10)");
    fmt.Println(" -t <timeout>       time out of per request (default 50 ms)");
    fmt.Println(" -D <duration>      test time duration for requests (default 5s)");
    fmt.Println(" -k <boolean>       true = keep alive, false = reconnect (default true)");
    fmt.Println(" -H                 show help information\n");
}

func checkParams(q int64, c int64) bool{
    if (q != 0 && c != 0) || (q == 0 && c == 0) {
        //qps和concurrency不能同时为0，或者同时不为0
        log.Logger.Fatal("The argu 'c' and 'q' can't be set at the same time -!\n\nRun the cmd: 'unicorn -H' for help!")
        return false
    }
    return true
}

//打印测试报告
func showReport() {

    //打印汇总结果
    var total int
    t.Log("Code Count:")
    for k, v := range count_map {
        code_plain := unicorn.ConvertCodePlain(k)
        t.Logf("  Code plain: %s (%d), Count: %d.\n", code_plain, k, v)
        total += v
    }

    //打印最终结果
    t.Logf("Total cnt: %d.\n", cnt)
    t.Logf("Total load: %d.\n", total)
    success_cnt := count_map[unicorn.RESULT_CODE_SUCCESS]
    tps := float64(success_cnt) / float64(duration/time.Second)
    t.Logf("Qps: %d; Tps(Treatments per second): %f.\n", qps, tps)

    /*
    float qps;
    char buf[11]; int idx=0, r; memset(buf, ' ', 11);

    qps = (float)g_conf.requests_done / ((float)g_conf.total_latency / 1000);
    r = (int)qps;

    fprintf(stdout, "============== %s REPORT ==============\n", g_conf.title);
    fprintf(stdout, "*   All requests           : %-10d               *\n", g_conf.requests);
    fprintf(stdout, "*   All requests sended    : %-10d               *\n", g_conf.requests_sended);
    fprintf(stdout, "*   All requests completed : %-10d               *\n", g_conf.requests_done);
    //如果done_if_srv_close，则done和finished相同，所以只有非done_if_srv_close时打印
    if(!g_conf.done_if_srv_close) fprintf(stdout, "*   All requests finished  : %-10d               *\n", g_conf.requests_finished);
    fprintf(stdout, "*   Use time of seconds    : %-10.2f               *\n", (float)g_conf.total_latency/1000);
    fprintf(stdout, "*   Parallel clients       : %-10d               *\n", g_conf.num_clients);
    fprintf(stdout, "*   Keep alive             : %-10d               *\n", g_conf.keep_alive);
    fprintf(stdout, "*                                                     *\n");
    fprintf(stdout, "*   Complete rate          : %-6.2f %%                 *\n", 100*((float)g_conf.requests_done/(float)g_conf.requests));
    while(r>0) { idx++; r /= 10; } idx = 10-idx; buf[idx] = '\0'; //计算最后一行对齐
    fprintf(stdout, "*   Average QPS            : %.2f r/s        %s*\n", qps, buf);

    */
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())
    //解析参数
    flag.Parse()
    if *H  {
        showUseage()
        return
    }

    //校验
    if !checkParams(int64(*q), int64(*c)) {
        return
    }

    address := fmt.Sprintf("%s:%s", *ip, *port)

    //初始化Plugin
    plg := plugin.NewTcpEquationPlugin()

    //初始化Unicorn
    result_chan := make(chan *unicorn.CallResult, 100) //结果回收通道
    timeout     := t*time.Millisecond                  //超时
    qps         := uint32(q)                           //期望的qps
    duration    := D * time.Second                     //探测持续时间
    concurrency := uint32(c)                           //并发度

    unc, err := unicorn.NewUnicorn(address, plg, timeout, qps, duration, concurrency, k, result_chan)
    if err != nil {
        log.Logger.Fatal(fmt.Sprintf("Unicorn initialization failing: %s.\n",  err))
        return
    }

    //开始干活儿! Start可以立刻返回的，进去看就知道~
    log.Logger.Info(fmt.Sprintf("Unicorn Start(timeout=%v, qps=%d, duration=%v)...", timeout, qps, duration))
    wg := unc.Start()

    //主流程在外面做一些总体控制工作，比如，循环阻塞接收结果~
    count_map := make(map[unicorn.ResultCode]int) //将结果按Code分类收集
    for ret := range result_chan {
        count_map[ret.Code] ++
        if v && ret.Code != unicorn.RESULT_CODE_SUCCESS{
            time := fmt.Sprintf(time.Now().Format("15:04:05"))
            log.Logger.Warning(fmt.Sprintf("[%s] Result: Id=%d, Code=%d, Msg=%s, Elapse=%v.\n", time, ret.Id, ret.Code, ret.Msg, ret.Elapse))
        }
    }

    wg.Wait()
}
