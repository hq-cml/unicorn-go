package lib

import (
    "fmt"
)

type MyLogger struct{}

func (MyLogger) Info(str interface{}){
    fmt.Println(str)
}

var Logger MyLogger

func init() {
    Logger = MyLogger{}
}