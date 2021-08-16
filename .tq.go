package tq

import (
	"sync"
	"time"
)

type TQ struct {

	// 达到任务执行时间时返回对应的Ts.P; 请及时读取, 否则会阻塞以致影响后续任务
	MQ chan interface{}
	// 赋值True将销毁队列实例; MQ不在有返回, 处理任务的协程将退出, Add将会painc
	Close bool

	addChan   chan Ts             // 增加任务管道
	taskChans map[int64](chan Ts) // map的Value(管道)存放任务, Key(int64)是任务管道的id
	endTimes  map[int64]time.Time // 记录对应任务管道的最后任务的执行时间
	idsChan   chan int64          // 传递id，表示新建了管道
	lock      sync.Mutex          // 互斥锁, 确保
}

// Ts 表示一个任务
type Ts struct {
	T time.Time   // 预定执行UTC时间
	P interface{} // 执行时返回的数据
}

func NewTQ() *TQ {
	var t = new(TQ)
	t.run()
	return t
}

// Run 启动
func (t *TQ) run() {
	t.addChan = make(chan Ts, 128)
	t.MQ = make(chan interface{}, 128)
	t.idsChan = make(chan int64, 64)
	t.taskChans = make(map[int64](chan Ts))
	t.endTimes = make(map[int64]time.Time)

	// exec新的taskChans
	go func() {
		for !t.Close {
			id := <-t.idsChan
			go t.exec(t.taskChans[id], id) // 执行每个管道中的任务
		}
	}()

	// 分发任务
	go func() {
		var r Ts
		var flag bool
		var chanId int64 // 管道id
		for !t.Close {
			r = <-t.addChan

			flag = false
			for id, v := range t.endTimes {
				if r.T.After(v) { //&& len(t.taskChans[id]) <= 1048576 追加
					t.taskChans[id] <- r
					t.endTimes[id] = r.T
					flag = true
					break
				}
			}

			// 需要新建管道
			if !flag {
				var tc chan Ts = make(chan Ts, 16)

				// 新建
				t.lock.Lock()
				t.taskChans[chanId] = tc // add
				t.lock.Unlock()
				t.endTimes[chanId] = r.T

				// 写入
				t.taskChans[chanId] <- r
				t.idsChan <- chanId

				chanId++
			}
		}

		close(t.addChan)
	}()
}

// Add 增加任务
func (t *TQ) Add(r Ts) error {
	// if cap(t.addChan)-len(t.addChan) < 1 {
	// 	return errors.New("blocked!")
	// }
	t.addChan <- r
	return nil
}

// Drop 结束任务队列
func (t *TQ) Drop() {
	t.Close = true
}

// exec 执行任务
func (t *TQ) exec(c chan Ts, id int64) {
	var ts Ts
	for !t.Close {
		if id != 0 && len(c) == 0 {
			// 释放
			t.lock.Lock()
			delete(t.endTimes, id)
			delete(t.taskChans, id)
			t.lock.Unlock()
			return
		}

		ts = <-c
		time.Sleep(time.Until(ts.T)) //延时
		if !t.Close {
			t.MQ <- ts.P
		}
	}
}
