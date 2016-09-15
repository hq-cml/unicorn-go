package plugin
/*
 * plugin
 * Tcp版本插件
 * 发送一个算式给服务端，服务端计算之后将结果返回
 */

import (
    "fmt"
    "github.com/hq-cml/unicorn-go/unicorn"
    "time"
    "encoding/json"
    "net"
    "bytes"
    "bufio"
    "strconv"
    "errors"
    "sync"
    "runtime"
    "math/rand"
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
    addr string
}

//*TcpEquationPlugin实现PluginIntfs接口
func (tep *TcpEquationPlugin) GenRequest() unicorn.RawRequest {
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
    raw_reqest := unicorn.RawRequest{Id: id, Req: bytes}
    return raw_reqest
}

func (tep *TcpEquationPlugin) Call(req []byte, timeout time.Duration) ([]byte, error) {
    conn, err := net.DialTimeout("tcp", tep.addr, timeout)
    if err != nil {
        return nil, err
    }

    _, err = write(conn, req, DELIM)
    if err != nil {
        return nil, err
    }
    return read(conn, DELIM)
}

func (tep *TcpEquationPlugin) CheckResponse(raw_req unicorn.RawRequest, raw_resp unicorn.RawResponse) *unicorn.CallResult {
    var result unicorn.CallResult
    result.Id = raw_resp.Id
    result.Req = raw_req
    result.Resp = raw_resp

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
    err = json.Unmarshal(raw_resp.Resp, &sresp)
    if err != nil {
        result.Code = unicorn.RESULT_CODE_ERROR_RESPONSE
        result.Msg = fmt.Sprintf("Incorrectly formatted Resp: %s!\n", string(raw_resp.Resp))
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
        result.Msg = fmt.Sprintf("Incorrect result: %s!\n",
            genFormula(sreq.Operands, sreq.Operator, sresp.Result, false))
        return &result
    }

    //一切都ok，则算是一次完整的请求
    result.Code = unicorn.RESULT_CODE_SUCCESS
    result.Msg = fmt.Sprintf("Success.(%s)", sresp.Formula)
    return &result
}

//New函数，创建TcpEquationPlugin，它是PluginIntfs的一个实现
func NewTcpEquationPlugin(addr string) unicorn.PluginIntfs {
    return &TcpEquationPlugin{
        addr: addr,
    }
}

//TODO 这两个函数应该挪到框架中去
func write(conn net.Conn, content []byte, delim byte) (int, error) {
    writer := bufio.NewWriter(conn)
    n, err := writer.Write(content)
    if err == nil {
        writer.WriteByte(delim)
    }
    if err == nil {
        err = writer.Flush()
    }
    return n, err
}

func read(conn net.Conn, delim byte) ([]byte, error) {
    readBytes := make([]byte, 1)
    var buffer bytes.Buffer
    for {
        _, err := conn.Read(readBytes)
        if err != nil {
            return nil, err
        }
        readByte := readBytes[0]
        if readByte == delim {
            break
        }
        buffer.WriteByte(readByte)
    }
    return buffer.Bytes(), nil
}

/**************************配套服务端的逻辑********************/
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

func genFormula(operands []int, operator string, result int, equal bool) string {
    var buff bytes.Buffer
    n := len(operands)
    for i := 0; i < n; i++ {
        if i > 0 {
            buff.WriteString(" ")
            buff.WriteString(operator)
            buff.WriteString(" ")
        }

        buff.WriteString(strconv.Itoa(operands[i]))
    }
    if equal {
        buff.WriteString(" = ")
    } else {
        buff.WriteString(" != ")
    }
    buff.WriteString(strconv.Itoa(result))
    return buff.String()
}

func reqHandler(conn net.Conn) {
    var errMsg string
    var sresp ServerEquationResp
    req, err := read(conn, DELIM)
    if err != nil {
        errMsg = fmt.Sprintf("Server: Req Read Error: %s", err)
    } else {
        var sreq ServerEquationReq
        err := json.Unmarshal(req, &sreq)
        if err != nil {
            errMsg = fmt.Sprintf("Server: Req Unmarshal Error: %s", err)
        } else {
            sresp.Id = sreq.Id
            sresp.Result = op(sreq.Operands, sreq.Operator)
            sresp.Formula =
            genFormula(sreq.Operands, sreq.Operator, sresp.Result, true)
        }
    }
    if errMsg != "" {
        sresp.Err = errors.New(errMsg)
    }
    bytes, err := json.Marshal(sresp)
    if err != nil {
        fmt.Errorf("Server: Resp Marshal Error: %s", err)
    }
    _, err = write(conn, bytes, DELIM)
    if err != nil {
        fmt.Errorf("Server: Resp Write error: %s", err)
    }
}

type TcpServer struct {
    listener net.Listener
    active   bool
    lock     *sync.Mutex
}

func (self *TcpServer) init(addr string) error {
    self.lock.Lock()
    defer self.lock.Unlock()
    if self.active {
        return nil
    }
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        return err
    }
    self.listener = ln
    self.active = true
    return nil
}

func (self *TcpServer) Listen(addr string) error {
    err := self.init(addr)
    if err != nil {
        return err
    }
    go func(active *bool) {
        for {
            conn, err := self.listener.Accept()
            if err != nil {
                fmt.Errorf("Server: Request Acception Error: %s\n", err)
                continue
            }
            go reqHandler(conn)
            runtime.Gosched()
        }
    }(&self.active)
    return nil
}

func (self *TcpServer) Close() bool {
    self.lock.Lock()
    defer self.lock.Unlock()
    if self.active {
        self.listener.Close()
        self.active = false
        return true
    } else {
        return false
    }
}

func NewTcpServer() *TcpServer {
    return &TcpServer{lock: new(sync.Mutex)}
}