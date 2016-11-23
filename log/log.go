package log
/*
 * 日志相关
 */
import (
    "fmt"
    "time"
)

type MyLogger struct{}

func (MyLogger) Info(str interface{}){
    time := fmt.Sprintf(time.Now().Format("2006-01-02 15:04:05"))
    fmt.Printf("[INFO]["+time+"] %v\n", str)
}

func (MyLogger) Warning(str interface{}){
    time := fmt.Sprintf(time.Now().Format("2006-01-02 15:04:05"))
    fmt.Printf("[WARN]["+time+"] %v\n", str)
}

func (MyLogger) Fatal(str interface{}){
    time := fmt.Sprintf(time.Now().Format("2006-01-02 15:04:05"))
    fmt.Printf("[FATAL]["+time+"] %v\n", str)
}

var Logger MyLogger

func init() {
    fmt.Println("Init log success")
    Logger = MyLogger{}
}