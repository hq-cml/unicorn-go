package plugin
/*
 * plugin
 * Tcp版本插件
 */

import (
    "fmt"
    "github.com/hq-cml/unicorn-go/unicorn"
    "time"
    "encoding/json"
    "net"
    "bytes"
    "bufio"
    "crypto/tls"
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
    Formula string   //公式
    Result  int      //结果
    Err     error
}

type TcpEquationPlugin struct {
    addr string
}

//*TcpEquationPlugin实现PluginIntfs接口
func (tep *TcpEquationPlugin) GenRequest() unicorn.RawReqest {
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
    raw_reqest := unicorn.RawReqest{Id: id, Req: bytes}
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