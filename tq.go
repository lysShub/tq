package tq

import (
	"sync"
	"time"
)

type TQ struct {

	// 达到任务执行时间时返回对应的Ts.P; 请及时读取, 否则会阻塞以致影响后续任务
	MQ chan interface{}

	addChan   chan Ts             // 任务增加管道
	taskChans map[int64](chan Ts) // map的Value(管道)存放任务, Key(int64)是任务管道的id
	endTimes  map[int64]time.Time // 记录对应任务管道的最后任务的执行时间
	idsChan   chan int64          // 传递id，表示新建了管道
	lock      sync.Mutex          // 读写锁
	tmpId     int64
}

// Ts 表示一个任务
type Ts struct {
	T time.Time   // 预定执行UTC时间
	P interface{} // 执行时返回的数据
}

// Run 启动
func (t *TQ) Run() {
	t.addChan = make(chan Ts, 128)
	t.MQ = make(chan interface{}, 128)
	t.idsChan = make(chan int64, 64)
	t.taskChans = make(map[int64](chan Ts))
	t.endTimes = make(map[int64]time.Time)

	// 执行
	go func() {
		for {
			id := <-t.idsChan
			go t.exec(t.taskChans[id], id) // 执行每个管道中的任务
		}
	}()

	// 分发任务
	go func() {
		var r Ts
		for {
			r = <-t.addChan

			var flag bool = false
			for id, v := range t.endTimes {

				if r.T.After(v) && len(t.taskChans[id]) <= 65536 { //追加
					t.taskChans[id] <- r
					t.endTimes[id] = r.T
					flag = true
					break
				}
			}
			// 需要新建管道
			if !flag {
				var sc chan Ts = make(chan Ts, 64*2)

				t.tmpId++
				if len(t.endTimes) == 0 {
					t.tmpId = 0
				}
				t.taskChans[t.tmpId] = sc // add

				t.endTimes[t.tmpId] = r.T
				t.taskChans[t.tmpId] <- r
				t.idsChan <- t.tmpId
			}
		}
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

// exec 执行任务
func (t *TQ) exec(c chan Ts, id int64) {
	var ts Ts
	for {

		t.lock.Lock()
		if id != 0 && len(c) == 0 {
			// 释放
			delete(t.endTimes, id)  // 删除endTimes中记录
			close(c)                // 关闭管道
			delete(t.taskChans, id) // 删除chans中记录
			t.lock.Unlock()
			return
		}
		t.lock.Unlock()

		ts = <-c
		time.Sleep(time.Until(ts.T)) //延时

		t.MQ <- ts.P
	}
}
