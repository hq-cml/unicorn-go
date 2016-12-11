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
    "github.com/hq-cml/reversi/ai"
)

type ReversiStatus int8

const (
    REVERSI_STATUS_ORIGIN     ReversiStatus = iota   //初始状态，这个状态下，client应该上报姓名
    REVERSI_STATUS_PUSH_NAME                         //已上报了自己的名字
    REVERSI_STATUS_PLACING                           //对弈过程中，循环确定落子
    REVERSI_STATUS_DONE                              //对弈结束
)

type TcpReversiPlugin struct {
    status     ReversiStatus       //当前状态
    role       int                 //本方是黑子还是白子
    myTurn     bool                //表示是否轮到己方落子
    chessBoard reversi.ChessBoard  //当前全局变量棋盘
}

//*TcpReversiPlugin实现PluginIntfs接口
//生成请求
func (trp *TcpReversiPlugin) GenRequest(id int64) unicorn.RawRequest {
    var msg string
    //状态机
    switch trp.status{
        case REVERSI_STATUS_ORIGIN:
            msg = "Nunicorn"  //上报姓名
            trp.status = REVERSI_STATUS_PUSH_NAME
        case REVERSI_STATUS_PUSH_NAME:
            fmt.Println("Something wrong!")
            os.Exit(1)
        case REVERSI_STATUS_PLACING:
            if trp.myTurn {
                //分析棋局，首先查看是否可落子
                step , canDown := ai.CheckChessboard(trp.chessBoard, int8(trp.role))
                if step == 0 {
                    fmt.Println("目前没有位置可落子，等待对方落子。。。")
                    msg = ""//返回空字符表示本次交互仍然是等待服务端返回数据
                } else {
                    //Ai落子
                    row, col := ai.AiPlayStep(trp.chessBoard, canDown, int8(trp.role))
                    msg = helper.ConvertRowColToServerProtocal(row, col)
                    fmt.Printf("AI(%d)落子：%d,%d, cmd:%s\n", trp.role, row, col, msg)
                    trp.myTurn = false
                }
            }else{
                //返回空字符表示本次交互仍然是等待服务端返回数据
                //fmt.Println("HAHA")
                msg = ""
            }
        case REVERSI_STATUS_DONE:
            fmt.Println("Something wrong!~")
            os.Exit(1)
    }

    raw_reqest := unicorn.RawRequest{Id: id, Req: []byte(msg)}
    return raw_reqest
}

//check服务端返回是否能够构成一个完整包
func (trp *TcpReversiPlugin)CheckFull(raw_req *unicorn.RawRequest, response []byte)(unicorn.ServerRespStatus) {
    //状态机
    switch trp.status{
        case REVERSI_STATUS_ORIGIN:
            return unicorn.SER_ERROR //不可能出现这种情况，因为一开始就会上报姓名
        case REVERSI_STATUS_PUSH_NAME:
            //上报姓名之后，服务端应该返回U1\n或者U0\n表示本方是哪一种棋子
            l := len(response)
            if l < 3 {
                return unicorn.SER_NEEDMORE
            } else if l > 3 && l < 69 {
                return unicorn.SER_NEEDMORE
            } else if l==3 || l==69 { //69是因为有的时候服务端会将首局棋盘一起发送过来
                return unicorn.SER_OK
            }

            return unicorn.SER_ERROR
        case REVERSI_STATUS_PLACING:
            //穷举对弈过程中的种种情况
            l := len(response)
            if l == 3 && string(response[0:2]) == "W1" {
                //You win!
                return unicorn.SER_OK
            } else if l == 3 && string(response[0:2]) == "W0" {
                //You lose!
                return unicorn.SER_OK
            } else if l == 3 && string(response[0:2]) == "W2" {
                //Draw tie!
                return unicorn.SER_OK
            } else if l == 2 && string(response[0:1]) == "G" {
                //Game over!
                return unicorn.SER_OK
            } else if l == 66 && string(response[0:1]) == "B"{
                //中间棋局
                return unicorn.SER_OK
            } else if string(response[0:1]) == "B" && l <66 {
                return unicorn.SER_NEEDMORE
            } else {
                return unicorn.SER_ERROR
            }
        case REVERSI_STATUS_DONE:
            fmt.Println("Something wrong!!")
            os.Exit(1)
    }

    return unicorn.SER_ERROR
}

//校验服务端返回是否符合预期
func (trp *TcpReversiPlugin) CheckResponse(raw_req unicorn.RawRequest, response []byte) (code unicorn.ResultCode, msg string) {
    //状态机
    switch trp.status{
        case REVERSI_STATUS_ORIGIN:
            code = unicorn.RESULT_CODE_ERROR_RESPONSE //不可能出现这种情况
        case REVERSI_STATUS_PUSH_NAME:
            if string(response[0:2]) == "U1" {
                fmt.Println("AI：黑子")
                trp.role = reversi.BLACK
                code = unicorn.RESULT_CODE_SUCCESS
            }else if string(response[0:2]) == "U0" {
                fmt.Println("AI：白子")
                trp.role = reversi.WIITE
                code = unicorn.RESULT_CODE_SUCCESS
            }else{
                code = unicorn.RESULT_CODE_ERROR_RESPONSE
            }

            //存在一种特殊情况：U1和棋盘放在一个TCP包中发过来了
            l := len(response)
            if l == 69 {
                fmt.Println("Got->",string(response[4:l]))
                //棋盘保存于全局变量
                trp.chessBoard = helper.ConverBytesToChessBoard(response[4:l-1])
                //打印棋盘
                reversi.PrintChessboard(trp.chessBoard)
                //轮到本方落子
                trp.myTurn = true
                code = unicorn.RESULT_CODE_SUCCESS
            }

            //棋盘已经保存在了全局变量，将状态变成PLACING
            trp.status = REVERSI_STATUS_PLACING
        case REVERSI_STATUS_PLACING:
            //穷举对弈过程中的种种情况
            l := len(response)
            if l == 3 && string(response[0:2]) == "W1" {
                fmt.Println("Got->",string(response[0:l-1]), ". [ You win! ]")
                code = unicorn.RESULT_CODE_SUCCESS
            } else if l == 3 && string(response[0:2]) == "W0" {
                fmt.Println("Got->",string(response[0:l-1]), ". [ You lose! ]")
                code = unicorn.RESULT_CODE_SUCCESS
            } else if l == 3 && string(response[0:2]) == "W2" {
                fmt.Println("Got->",string(response[0:l-1]), ". [ Draw tie! ]")
                code = unicorn.RESULT_CODE_SUCCESS
            } else if l == 2 && string(response[0:1]) == "G" {
                fmt.Println("Got->",string(response[0:l-1]), ". [ Game over! ]")
                //RESULT_CODE_DONE，通知框架结束程序！！
                code = unicorn.RESULT_CODE_DONE
                trp.status = REVERSI_STATUS_DONE //棋局完成状态
                //os.Exit(0)
            } else if l == 66 && string(response[0:1]) == "B"{
                fmt.Println("Got->",string(response[0:l]))
                //棋盘保存于全局变量
                trp.chessBoard = helper.ConverBytesToChessBoard(response[1:l-1])
                //打印棋盘
                reversi.PrintChessboard(trp.chessBoard)
                //轮到本方落子
                trp.myTurn = true
                code = unicorn.RESULT_CODE_SUCCESS
            }  else {
                code = unicorn.RESULT_CODE_ERROR_RESPONSE
            }
        case REVERSI_STATUS_DONE:
            fmt.Println("Something wrong!!!")
            os.Exit(1)
    }

    return
}

//New函数，创建TcpReversiPlugin，它是PluginIntfs的一个实现
func NewTcpReversiPlugin() unicorn.PluginIntfs {
    return &TcpReversiPlugin{
        status    : REVERSI_STATUS_ORIGIN,
        role      : reversi.BLACK,          //默认本方是黑子
        myTurn    : false,                  //非本方落子
    }
}
