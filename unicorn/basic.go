package unicorn
/*
 * 基础类型定义
 */

import "time"

//原生request的结构。本质上就是字节流
type RawReqest struct {
    Id  int64   //请求Id，这个request应该一一对应
    Req []byte  //字节流
}

//原生response结构。出了字节流之外，还有错误标记和耗时
type RawResponse struct {
    Id      int64          //请求Id，这个request应该一一对应
    Resp    []byte         //字节流
    Err     error          //是否错误
    Elapse  time.Duration  //请求耗时
}

type ResultCode int

//调用结果的结构。
type CallResult struct {
    Id      int64         //ID
    Req     RawReqest     //原生请求
    Resp    RawResponse   //原生响应
    Code    ResultCode    //响应码
    Msg     string        //细节信息
    Elapse  time.Duration //耗时
}

//unicorn的当前状态
type UncStatus int
const (
    ORIGINAL UncStatus = iota  //0
    STARTED                    //1
    STOPPED                    //2
)

const (
    RESULT_CODE_SUCCESS          = 0    //成功
    RESULT_CODE_WARING_TIMEOUT   = 1001 //请求超时
    RESULT_CODE_ERROR_CALL       = 2001 //错误调用
    RESULT_CODE_ERROR_RESPONSE   = 2002 //错误的相应内容
    RESULT_CODE_ERROR_CALEE      = 2003 //被调用方内部错误
    RESULT_CODE_FATAL_CALL       = 3001 //调用过程中的致命错误
)

func ConvertCodePlain(code ResultCode) string {
    var code_plain string
    switch code {
    case RESULT_CODE_SUCCESS:
        code_plain = "Success"
    case RESULT_CODE_WARNING_CALL_TIMEOUT:
        code_plain = "Call Timeout Warning"
    case RESULT_CODE_ERROR_CALL:
        code_plain = "Call Error"
    case RESULT_CODE_ERROR_RESPONSE:
        code_plain = "Response Error"
    case RESULT_CODE_ERROR_CALEE:
        code_plain = "Callee Error"
    case RESULT_CODE_FATAL_CALL:
        code_plain = "Call Fatal Error"
    default:
        code_plain = "Unknown result code"
    }
    return code_plain
}
//插件接口，实现这个接口，嵌入unicorn，即可组成完整的客户端
type PluginIntfs interface {
    //构造请求
    GenRequest() RawReqest
    //调用
    Call(req []byte, timeout time.Duration)([]byte, error)
    //检查响应
    CheckResponse(rawReq RawReqest, rawResp RawResponse) *CallResult
}