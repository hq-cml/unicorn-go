package plugin
/*
 * plugin
 * Tcp版本插件
 */

import (
    "fmt"
    "github.com/hq-cml/unicorn-go/unicorn"
)

const (
    DELIM = '\n'
)

type TcpPlugin strct {
    addr string
}

//New函数，创建TcpPlugin
func NewTcpPlugin(addr string) unicorn.PluginIntfs {

}

//*TcpPlugin实现PluginIntfs接口
