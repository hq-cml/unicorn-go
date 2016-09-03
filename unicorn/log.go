package unicorn
/*
 * 日志相关
 */
import (
    "fmt"
)

type MyLogger struct{}

func (MyLogger) Info(str interface{}){
    fmt.Println(str)
}

var Logger MyLogger

func init() {
    fmt.Println("Init log success")
    Logger = MyLogger{}
}