package main

import (
    //"github.com/hq-cml/unicorn-go/log"
    //"github.com/hq-cml/unicorn-go/unicorn"
    "runtime"
    "flag"
    "fmt"
    //"errors"
)

var ip *string = flag.String("h", "127.0.0.1", "ip")
var port *string = flag.String("p", "9527", "port")
var mode *int = flag.Int("a", 0, "AI")                  //0-手动模式 1-AI自动模式

func showUseage() {
    fmt.Println()
    fmt.Println("Usage: ./Unicorn [-h <ip>] [-p <port>] [-c <concurrency>] [-D duration]> [-k <boolean>]");
    fmt.Println("Note: !!!!- The argu 'c' and 'q' can not set at the same time -!!!!");
    fmt.Println()
    fmt.Println(" -h <hostname>      server hostname (default 127.0.0.1)");
    fmt.Println(" -p <port>          server port (default 9527)");
    fmt.Println(" -c <concurrency>   number of parallel connections (default 1)");
    fmt.Println(" -q <qps>           qps-- the frequence you wanted for requests (default 10)");
    fmt.Println(" -D <duration>      total duration of requests (default 5s)");
    fmt.Println(" -k <boolean>       1 = keep alive, 0 = reconnect (default 1)");
    fmt.Println(" -H                 show help information\n");
}

func checkParams() {

}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())
    showUseage()
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
