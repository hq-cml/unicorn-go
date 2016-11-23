package unicorn
/*
 * 基础类型定义
 */
import (
    "time"
    wp "github.com/hq-cml/unicorn-go/worker-pool"
)

//Unicorn接口
type UnicornIntfs interface {
    Start()                //启动unicorn
    Stop() (uint64, bool)  //第一个返回值表示停止时已完成请求数，第二个返回值表示是否成功停止
    Status() UncStatus     //获得unicorn当前状态
}

//Unicorn接口的实现类型
type Unicorn struct {
    qps           uint32             //规定每秒的请求量
    timeout       time.Duration      //规定的每个请求最大延迟
    duration      time.Duration      //持续探测访问持续时间
    concurrency   uint32             //并发量，这个值是根据timeout和qps算出来的
    sigChan       chan byte          //信号指令接收通道，Unicorn通过这个通道接收指令，比如Stop指令
    status        UncStatus          //当前状态
    resultChan    chan *CallResult   //保存调用结果的通道
    plugin        PluginIntfs        //插件接口，提供扩展功能，用户实现Plugin接口，嵌入Unicorn框架即可实现自己的client
    pool          wp.WorkerPoolIntfs //goroutine协程池，控制并发量
    stopFlag      bool               //停止发送后续结果的标记。
    //finalCnt      chan uint64      //完结信号的传递通道，同时被用于传递调用执行计数。感觉这样的设计完全没必要
    finalCnt      uint64             //最终调用计数
}

//原生request的结构。本质上就是字节流
type RawRequest struct {
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

//结果码
type ResultCode int
const (
    RESULT_CODE_SUCCESS         ResultCode = 0    //成功
    RESULT_CODE_WARING_TIMEOUT  ResultCode = 1001 //请求超时
    RESULT_CODE_ERROR_CALL      ResultCode = 2001 //错误调用
    RESULT_CODE_ERROR_RESPONSE  ResultCode = 2002 //错误的相应内容
    RESULT_CODE_ERROR_CALEE     ResultCode = 2003 //被调用方内部错误
    RESULT_CODE_FATAL_CALL      ResultCode = 3001 //调用过程中的致命错误
)

//调用结果的结构。
type CallResult struct {
    Id     int64         //ID
    Req    RawRequest    //原生请求
    Resp   RawResponse   //原生响应
    Code   ResultCode    //响应码
    Msg    string        //细节信息
    Elapse time.Duration //耗时，这个貌似和RawResponse里面的Elapse重复。。
}

//unicorn的当前状态
type UncStatus int
const (
    ORIGINAL UncStatus = iota  //0
    STARTED                    //1
    STOPPED                    //2
)

//插件接口，实现这个接口，嵌入unicorn，即可组成完整的客户端
type PluginIntfs interface {
    //构造请求
    GenRequest() RawRequest
    //调用
    Call(req []byte, timeout time.Duration)([]byte, error)
    //检查响应
    CheckResponse(rawReq RawRequest, rawResp RawResponse) *CallResult
}