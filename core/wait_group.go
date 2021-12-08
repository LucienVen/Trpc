package core

// TODO 未实现

/**
WaitGroupCore
counter 当前尚未结束执行的计数器
waiter 等待goroutine-group结束的goroutine数量，即有多少个等待者
semaphore 信号量，
 */
type WaitGroupCore struct {
	stata1 [3]uint32
}

// state 返回指向 wg.state1 中存储的 state 和 sema 字段的指针。
func (w *WaitGroupCore) state()  {

}

// 1. 把delta值加在counter上
// 2. 当counter 变为0时，根据waiter数量释放信号量，把等待的goroutine全部唤醒
// 3. 当counter 为负数时，panic
func (w *WaitGroupCore) Add(delta int)  {

}

// waiter 递增1，并阻塞信号量
func (w *WaitGroupCore) Wait()  {

}

// counter 递减1，并按照waiter数值释放信号量
func (w *WaitGroupCore) Done()  {

}
