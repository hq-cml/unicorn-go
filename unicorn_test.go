package main

import (
    "testing"
    "runtime"
    "github.com/hq-cml/unicorn-go/unicorn"
    "github.com/hq-cml/unicorn-go/plugin"
)

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
}
