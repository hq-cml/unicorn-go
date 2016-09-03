package unicorn
/*
 * goroutine协程池实现
 * 利用缓冲通道，实现一个goroutine池子
 */

//goroutine协程池接口
type WorkerPoolIntfs interface {
    Take()             //产生一个goroutine
    Return()           //一个goroutine结束
    Active() bool      //池子是否激活状态
    Total() uint32     //最大goroutine数量
    Remainder() uint32 //池子中的剩余空闲goroutine
}

//接口的实现
type workerPool struct {
    total  uint32     //池子容量，即最大协程数量
    pool   chan byte  //容器，利用带缓冲通道实现
    active bool       //是否激活标记
}

//*workerPool将会实现接口WorkerPoolIntfs
func (wp *workerPool) Take() {
    <- wp.pool
}
func (wp *workerPool) Return() {
    wp.pool <- 1
}
func (wp *workerPool) Active() bool {
    return wp.active
}
func (wp *workerPool) Total() uint32 {
    return wp.total
}
func (wp *workerPool) Remaider() uint32 {
    //每取走一个goroutine，会从通道中取走一个元素
    //搜易通道的长度就是剩余goroutine的数量
    return uint32(len(wp.pool))
}

//初始化
func (wp *workerPool) create(total uint32) bool {
    //if wp.active
}