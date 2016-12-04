package plugin
/*
 * plugin
 * Tcp版本插件
 * 黑白棋的客户端
 */

import (
    "fmt"
    "github.com/hq-cml/unicorn-go/unicorn"
    "github.com/hq-cml/reversi"
)

type ReversiStatus int8

const (
    REVERSI_ORIGIN  ReversiStatus = iota   //初始状态，这个状态下，client应该上报姓名
    REVERSI_PUSH_NAME                      //已上报了自己的名字
    REVERSI_PLACING                        //对弈过程中，循环确定落子
    REVERSI_DONE                           //对弈结束
)

type TcpReversiPlugin struct {
    status ReversiStatus  //当前状态
    role   int            //本方是黑子还是白子
}

//*TcpReversiPlugin实现PluginIntfs接口
//生成请求
func (tep *TcpReversiPlugin) GenRequest(id int64) unicorn.RawRequest {
    var msg string
    if (tep.status == REVERSI_ORIGIN) {
        msg = "Nhq"  //上报姓名
        tep.status = REVERSI_PUSH_NAME
    }


    raw_reqest := unicorn.RawRequest{Id: id, Req: []byte(msg)}
    return raw_reqest
}

//check服务端返回是否能够构成一个完整包
func (tep *TcpReversiPlugin)CheckFull(raw_req *unicorn.RawRequest, response []byte)(unicorn.ServerRespStatus) {
    if tep.status == REVERSI_PUSH_NAME {
        return unicorn.SER_ERROR //不可能出现这种情况，因为一开始就会上报姓名
    }else if tep.status == REVERSI_PUSH_NAME {
        if len(response) < 2 {
            return unicorn.SER_NEEDMORE
        }

        return unicorn.OK
        if string(response[0:2]) == "U1" {
            fmt.Println("AI：黑子")
            tep.role = reversi.BLACK
            return unicorn.SER_OK
        }else if string(response[0:2]) == "U0" {
            fmt.Println("AI：白子")
            tep.role = reversi.WIITE
            return unicorn.SER_OK
        }else{
            return unicorn.SER_ERROR
        }
    } else if tep.status == REVERSI_PLACING {

    } else {

    }

    len1 := len(raw_req.Req)
    len2 := len(response)

    //对于回显程序，长度相同则表示包符合预期
    if len1 == len2 {
        return unicorn.SER_OK
    } else if len1 > len2 {
        return unicorn.SER_NEEDMORE
    } else {
        return unicorn.SER_ERROR
    }
}

//校验服务端返回是否符合预期
func (tep *TcpReversiPlugin) CheckResponse(raw_req unicorn.RawRequest, response []byte) (code unicorn.ResultCode, msg string) {

    if tep.status == REVERSI_PUSH_NAME {
        //算是一种错误的返回，不可能出现这种情况
        code = unicorn.RESULT_CODE_ERROR_RESPONSE
    }else if tep.status == REVERSI_PUSH_NAME {
        if string(response[0:2]) == "U1" {
            fmt.Println("AI：黑子")
            tep.role = reversi.BLACK
            code = unicorn.RESULT_CODE_SUCCESS
        }else if string(response[0:2]) == "U0" {
            fmt.Println("AI：白子")
            tep.role = reversi.WIITE
            code = unicorn.RESULT_CODE_SUCCESS
        }else{
            return unicorn.SER_ERROR
        }
    } else if tep.status == REVERSI_PLACING {

    } else {

    }
    return
}

//New函数，创建TcpReversiPlugin，它是PluginIntfs的一个实现
func NewTcpReversiPlugin() unicorn.PluginIntfs {
    return &TcpReversiPlugin{
        status : REVERSI_ORIGIN,
        role: reversi.BLACK,      //默认本方是黑子
    }
}
