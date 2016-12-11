package unicorn

/*
 * 基础类型定义
 */
import (
    "time"
    wp "github.com/hq-cml/unicorn-go/worker-pool"
    "sync"
)

//Unicorn接口
type UnicornIntfs interface {
    Start() *sync.WaitGroup  //启动unicorn
    Stop() (uint64, bool)    //第一个返回值表示停止时已完成请求数，第二个返回值表示是否成功停止
    Status() UncStatus       //获得unicorn当前状态
}

//Unicorn接口的实现类型
type Unicorn struct {
    serverAdd   string             //服务端地址
    qps         uint32             //每秒的请求量，这个值和下面concurrency不同时设置，因为会存在一定的矛盾
    concurrency uint32             //并发量，这个值不能喝qps同时设置，可以用户指定，或者根据timeout和qps算出来
    timeout     time.Duration      //规定的每个请求最大延迟
    Duration    time.Duration      //持续探测访问持续时间
    sigChan     chan byte          //信号指令接收通道，Unicorn通过这个通道接收指令，比如Stop指令
    status      UncStatus          //当前状态
    resultChan  chan *CallResult   //保存调用结果的通道
    plugin      PluginIntfs        //插件接口，提供扩展功能，用户实现Plugin接口，嵌入Unicorn框架即可实现自己的client
    pool        wp.WorkerPoolIntfs //goroutine协程池，控制并发量
    stopFlag    bool               //停止发送后续结果的标记。
    AllCnt      uint64             //最终总的调用的计数
    IgnoreCnt   uint64             //忽略掉的请求的计数
    throttle    <-chan time.Time   //断续器（time.Tick），用来控制请求的频率，如果设置了qps，则断续器有效非空
    keepalive   bool               //是否维持长连接模式
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
    RESULT_CODE_ERROR_CALL      ResultCode = 2001 //请求发生错误
    RESULT_CODE_ERROR_RESPONSE  ResultCode = 2002 //错误的响应内容
    RESULT_CODE_ERROR_CALEE     ResultCode = 2003 //被调用方内部错误
    RESULT_CODE_FATAL_CALL      ResultCode = 3001 //调用过程中的致命错误
    //RESULT_CODE_DONE            ResultCode = 4001 //结束，框架应该断开连接
)

//调用结果的结构。
type CallResult struct {
    Id     int64         //ID
    //Req    RawRequest    //原生请求
    //Resp   RawResponse   //原生响应
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

//判断服务端响应是否完整的几种情况
type ServerRespStatus int8
const (
    SER_OK          ServerRespStatus = iota  //0 完整返回
    SER_NEEDMORE                             //1 没能找到合适的包定界符，还需要继续读取网络数据
    SER_ERROR                                //2 服务端出现了某种错误
)

//插件接口，实现这个接口，嵌入unicorn，即可组成完整的客户端
type PluginIntfs interface {
    //必选函数：生成请求内容
    //某些时候，处理了一个包，下一个包不一定要主动发送，而是需要被动等待
    //这种情况下，需要将RawRequest.Req设置为""
    GenRequest(id int64) RawRequest
    //必选函数：判断接收到的内容，是否是完整的响应包
    CheckFull(rawReq *RawRequest, response []byte)(ServerRespStatus)
    //必选函数：检查响应内容是否符合用户需求
    CheckResponse(rawReq RawRequest, response []byte) (ResultCode, string)
}