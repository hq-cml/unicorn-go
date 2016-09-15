package plugin
/*
 * plugin
 * Tcp版本插件
 */

import (
    "fmt"
    "github.com/hq-cml/unicorn-go/unicorn"
    "time"
    "encoding/json"
)

const (
    DELIM = '\n'
)

type ServerEquationReq struct {
    Id       int64
    Operands []int   //操作数
    Operator string  //操作符
}

type ServerEquationResp struct {
    Id      int64
    Formula string   //公式
    Result  int      //结果
    Err     error
}

type TcpEquationPlugin strct {
    addr string
}

//*TcpEquationPlugin实现PluginIntfs接口
func (tep *TcpEquationPlugin) GenRequest() unicorn.RawReqest {
    id := time.Now().UnixNano() //用纳秒就能保证唯一性了吗？
    req := ServerEquationReq{
        Id: id,
        Operands:[]int{ //两个随机数
            int(rand.Int31n(1000) + 1),
            int(rand.Int31n(1000) + 1),
        },
        Operator: func() string {
            op := []string{"+", "-", "*", "/"}
            return op[rand.Int31n(100)%4]
        }(),
    }
    bytes, err := json.Marshal(req)
    if err != nil {
        panic(err) //框架会接住这个panic，defer unc.handleError()
    }
    raw_reqest := unicorn.RawReqest{Id: id, Req: bytes}
    return raw_reqest
}

//New函数，创建TcpEquationPlugin，它是PluginIntfs的一个实现
func NewTcpEquationPlugin(addr string) unicorn.PluginIntfs {
    return &TcpEquationPlugin{
        addr: addr,
    }
}