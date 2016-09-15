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

type TcpEquationPlugin strct {
    addr string
}

//*TcpPlugin实现PluginIntfs接口


//New函数，创建TcpEquationPlugin，它是PluginIntfs的一个实现
func NewTcpEquationPlugin(addr string) unicorn.PluginIntfs {
    return &TcpEquationPlugin{
        addr: addr,
    }
}