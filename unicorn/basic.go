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

//插件接口，实现这个接口，嵌入unicorn，即可组成完整的客户端
type Plugin interface {
    //构造请求
    GenReq() RawReqest

    //调用
    Call(req []byte, timeout time.Duration)([]byte, error)

    //检查响应
    CheckResp(rawReq RawReqest, rawResp RawResponse) *CallResult
}