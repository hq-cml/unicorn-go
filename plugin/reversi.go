package plugin
/*
 * plugin
 * Tcp版本插件
 * 黑白棋的客户端
 */

import (
    "fmt"
    "github.com/hq-cml/unicorn-go/unicorn"
    "github.com/hq-cml/go-case/random"
)

type TcpReversiPlugin struct {
}

//*TcpEchoPlugin实现PluginIntfs接口
//生成请求
func (tep *TcpReversiPlugin) GenRequest(id int64) unicorn.RawRequest {
    //生成随机字符串，作为消息
    msg := random.GenRandString(10)

    raw_reqest := unicorn.RawRequest{Id: id, Req: []byte(msg)}
    return raw_reqest
}

//check服务端返回是否能够构成一个完整包
func (tep *TcpReversiPlugin)CheckFull(raw_req *unicorn.RawRequest, response []byte)(unicorn.ServerRespStatus) {
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
    str1 := string(raw_req.Req)
    str2 := string(response)

    if str1 == str2 {
        code = unicorn.RESULT_CODE_SUCCESS
        msg = fmt.Sprintf("Success.(%s)", string(response))
    } else {
        code = unicorn.RESULT_CODE_ERROR_RESPONSE
        msg = fmt.Sprintf("Incorrectly formatted Resp: %s!\n", string(response))
    }

    return
}

//New函数，创建TcpEquationPlugin，它是PluginIntfs的一个实现
func TcpReversiPlugin() unicorn.PluginIntfs {
    return &TcpReversiPlugin{ }
}
