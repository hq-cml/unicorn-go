package plugin
/*
 * plugin
 * Tcp版本插件
 * 发送一个算式给服务端，服务端计算之后将结果返回
 */

import (
    "fmt"
    "github.com/hq-cml/unicorn-go/unicorn"
    "encoding/json"
    //"net"
    //"bytes"
    //"bufio"
    "math/rand"
    //"strconv"
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
    Formula string   //具体公式
    Result  int      //结果
    Err     error
}

type TcpEquationPlugin struct {
}

//*TcpEquationPlugin实现PluginIntfs接口
func (tep *TcpEquationPlugin) GenRequest(id int64) unicorn.RawRequest {
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
    bytes = append(bytes, DELIM)
    raw_reqest := unicorn.RawRequest{Id: id, Req: bytes}
    return raw_reqest
}

func (tep *TcpEquationPlugin)CheckFull(id int64, response []byte)(unicorn.ServerRespStatus) {
    //校验response
    var sresp ServerEquationResp

    l := len(response)

    if response[l-1] == DELIM {
        err := json.Unmarshal(response[:l-1], &sresp)
        if err != nil {
            fmt.Println("AAAA")
            return unicorn.SER_NEEDMORE
        }

        if sresp.Id != id {
            return unicorn.SER_ERROR
        }
    } else {
        fmt.Println(string(response), len(response))
        return unicorn.SER_NEEDMORE
    }

    return unicorn.SER_OK
}

func (tep *TcpEquationPlugin) CheckResponse(raw_req unicorn.RawRequest, response []byte) *unicorn.CallResult {
    var result unicorn.CallResult

    //校验request
    var sreq ServerEquationReq
    err := json.Unmarshal(raw_req.Req, &sreq)
    if err != nil {
        result.Code = unicorn.RESULT_CODE_FATAL_CALL
        result.Msg = fmt.Sprintf("Incorrectly formatted Req: %s!\n", string(raw_req.Req))
        return &result
    }

    //校验response
    var sresp ServerEquationResp
    err = json.Unmarshal(response, &sresp)
    if err != nil {
        result.Code = unicorn.RESULT_CODE_ERROR_RESPONSE
        result.Msg = fmt.Sprintf("Incorrectly formatted Resp: %s!\n", string(response))
        return &result
    }

    //校验id是否一致
    if sresp.Id != sreq.Id {
        result.Code = unicorn.RESULT_CODE_ERROR_RESPONSE
        result.Msg = fmt.Sprintf("Inconsistent raw id! (%d != %d)\n", sresp.Id ,sreq.Id)
        return &result
    }

    //校验response的Err
    if sresp.Err != nil {
        result.Code = unicorn.RESULT_CODE_ERROR_CALEE
        result.Msg = fmt.Sprintf("Abnormal server: %s!\n", sresp.Err)
        return &result
    }

    //校验最终计算结果是否一致
    if sresp.Result != op(sreq.Operands, sreq.Operator) {
        result.Code = unicorn.RESULT_CODE_ERROR_RESPONSE
        result.Msg = fmt.Sprintf("Incorrect result: %s!\n", genFormula(sreq.Operands, sreq.Operator, sresp.Result, false))
        return &result
    }

    //一切都ok，则算是一次完整的请求
    result.Id = sresp.Id
    result.Code = unicorn.RESULT_CODE_SUCCESS
    result.Msg = fmt.Sprintf("Success.(%s)", sresp.Formula)
    return &result
}

//New函数，创建TcpEquationPlugin，它是PluginIntfs的一个实现
func NewTcpEquationPlugin() unicorn.PluginIntfs {
    return &TcpEquationPlugin{ }
}

func op(operands []int, operator string) int {
    var result int
    switch {
    case operator == "+":
        for _, v := range operands {
            if result == 0 {
                result = v
            } else {
                result += v
            }
        }
    case operator == "-":
        for _, v := range operands {
            if result == 0 {
                result = v
            } else {
                result -= v
            }
        }
    case operator == "*":
        for _, v := range operands {
            if result == 0 {
                result = v
            } else {
                result *= v
            }
        }
    case operator == "/":
        for _, v := range operands {
            if result == 0 {
                result = v
            } else {
                result /= v
            }
        }
    }
    return result
}

