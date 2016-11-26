package worker_pool

/*
 * goroutine协程池实现
 * 利用缓冲通道作为载体，实现一个goroutine池子，利用worker-pool可以控制goroutine最大数量
 *
 * 实现思路：
 * 票池，类似于POSIX的信号量。初始池子是满的，每生产一个goroutine，则向票池中取一张票;
 * 每消亡一个goroutine，就归还一张票。当票池为空的时候，所有生产goroutine的行为将阻塞
 */
import (
    "fmt"
    "errors"
)

//goroutine协程池接口
type WorkerPoolIntfs interface {
    Take()             //产生一个goroutine
    Return()           //一个goroutine结束
    Active() bool      //池子是否激活状态
    Total() uint32     //最大goroutine数量
    Remainder() uint32 //池子中的剩余空闲goroutine
}

//接口的实现
type WorkerPool struct {
    total  uint32     //池子容量，即最大协程数量
    pool   chan byte  //容器，利用带缓冲通道实现
    active bool       //是否激活标记
}

//*workerPool将会实现接口WorkerPoolIntfs
func (wp *WorkerPool) Take() {
    <- wp.pool
}
func (wp *WorkerPool) Return() {
    wp.pool <- 1
}
func (wp *WorkerPool) Active() bool {
    return wp.active
}
func (wp *WorkerPool) Total() uint32 {
    return wp.total
}
func (wp *WorkerPool) Remainder() uint32 {
    //每取走一个goroutine，会从通道中取走一个元素
    //所以通道的长度就是剩余goroutine的数量
    return uint32(len(wp.pool))
}

//实例化协程池，New开头的惯例
//返回值是WorkerPoolIntfs的实现，所以是*workerPool
func NewWorkerPool(total uint32) (WorkerPoolIntfs, error) {
    if total == 0 {
        msg := fmt.Sprintf("Worker Pool init Failed. total=%d", total)
        return nil, errors.New(msg)
    }

    wp := WorkerPool{}
    //初始化通道，带缓冲的！
    ch := make(chan byte, total)
    //将通道填满，表示票池是满的，即没有goroutine生产
    var i uint32
    for i=0; i<total; i++ {
        ch <- 1
    }
    wp.pool = ch
    wp.total = total
    wp.active = true

    return &wp, nil
}