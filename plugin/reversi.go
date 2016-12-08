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
    "github.com/hq-cml/reversi/client/helper"
    "os"
)

var chessBoard []byte //当前全局变量棋盘

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
    switch tep.status{
        case REVERSI_ORIGIN:
            msg = "Nhq"  //上报姓名
            tep.status = REVERSI_PUSH_NAME
        case REVERSI_PUSH_NAME:
            fmt.Println("Something wrong!")
            os.Exit(1)
        case REVERSI_PLACING
    }



    raw_reqest := unicorn.RawRequest{Id: id, Req: []byte(msg)}
    return raw_reqest
}

//check服务端返回是否能够构成一个完整包
func (tep *TcpReversiPlugin)CheckFull(raw_req *unicorn.RawRequest, response []byte)(unicorn.ServerRespStatus) {
    if tep.status == REVERSI_PUSH_NAME {
        return unicorn.SER_ERROR //不可能出现这种情况，因为一开始就会上报姓名
    }else if tep.status == REVERSI_PUSH_NAME {
        //上报姓名之后，服务端应该返回U1\n或者U0\n表示本方是哪一种棋子
        l = len(response)
        if l < 3 {
            return unicorn.SER_NEEDMORE
        } else if l > 3 && l < 69 {
            return unicorn.SER_NEEDMORE
        } else if l==3 || l==69 {
            return unicorn.OK
        }

        return unicorn.SER_ERROR

    } else if tep.status == REVERSI_PLACING {

    } else {

    }
}

//校验服务端返回是否符合预期
func (tep *TcpReversiPlugin) CheckResponse(raw_req unicorn.RawRequest, response []byte) (code unicorn.ResultCode, msg string) {

    if tep.status == REVERSI_PUSH_NAME {               //当前处于初始状态
        //算是一种错误的返回，不可能出现这种情况
        code = unicorn.RESULT_CODE_ERROR_RESPONSE
    }else if tep.status == REVERSI_PUSH_NAME {         //当前处于已上报姓名阶段
        if string(response[0:2]) == "U1" {
            fmt.Println("AI：黑子")
            tep.role = reversi.BLACK
            code = unicorn.RESULT_CODE_SUCCESS
        }else if string(response[0:2]) == "U0" {
            fmt.Println("AI：白子")
            tep.role = reversi.WIITE
            code = unicorn.RESULT_CODE_SUCCESS
        }else{
            code = unicorn.RESULT_CODE_ERROR_RESPONSE
        }

        //存在一种特殊情况：U1和棋盘放在一个TCP包中发过来了
        l := len(response)
        if l == 69 {
            //打印棋盘
            fmt.Println("Got->",string(response[4:l]))
            chessBoard = helper.ConverBytesToChessBoard(response[4:l-1])
            reversi.PrintChessboard(chessBoard)

            code = unicorn.RESULT_CODE_SUCCESS
        }

        //棋盘已经保存在了全局变量，将状态变成PLACING
        tep.status = REVERSI_PLACING
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
