package main

import (
    //"github.com/hq-cml/unicorn-go/log"
    //"github.com/hq-cml/unicorn-go/unicorn"
    "runtime"
    "flag"
    "fmt"
    "github.com/hq-cml/unicorn-go/log"
)

var ip *string = flag.String("h", "127.0.0.1", "ip")
var port *string = flag.String("p", "9527", "port")
var c *int = flag.Int("c", 1, "concurrency")
var q *int = flag.Int("q", 10, "qps")
var D *int = flag.Int("D", 5, "port")
var k *bool = flag.Bool("k", true, "keep alive")
var H *bool = flag.Bool("H", false, "help")

func showUseage() {
    fmt.Println()
    fmt.Println("Usage: unicorn [-h <ip>] [-p <port>] [-c <concurrency>] [-D duration]> [-k <boolean>]");
    fmt.Println("Note: !!!!- The argu 'c' and 'q' can't be set at the same time -!!!!");
    fmt.Println()
    fmt.Println(" -h <hostname>      server hostname (default 127.0.0.1)");
    fmt.Println(" -p <port>          server port (default 9527)");
    fmt.Println(" -c <concurrency>   number of parallel connections (default 1)");
    fmt.Println(" -q <qps>           qps-- the frequence you wanted for requests (default 10)");
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

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())
    //解析参数
    flag.Parse()
    if *H  {
        showUseage()
        return
    }

    if !checkParams(int64(*q), int64(*c)) {
        return
    }

    //提示输出
    //fmt.Println(`
    //Enter following commands to control:
    //Nyourname -- report your name. Nhq, eg.
    //Row&Col -- place pieces in (Row,Col). 3D, eg.
    //quit -- quit
    //`)
    //
    //
    ////参数校验
    //if (qps != 0 && concurrency != 0) ||
    //(qps == 0 && concurrency == 0) {
    //    //qps和concurrency不能同时为0，或者同时不为0
    //    return nil, errors.New("qps and concurrency can't be 0 all. or is 0 all!")
    //}
    //if plugin == nil {
    //    return nil, errors.New("Nil plugin")
    //}
    //if timeout == 0 {
    //    return nil, errors.New("Nil timeout")
    //}
    //if duration == 0 {
    //    return nil, errors.New("Nil duration")
    //}
    //if resultChan == nil {
    //    return nil, errors.New("Nil resultChan")
    //}
    //if addr == "" {
    //    return nil, errors.New("Nil address")
    //}
    //
    ////
    //address := fmt.Sprintf("%s:%s", *ip, *port)

    //fmt.Println("address:", address)
}
