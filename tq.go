package tq

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strconv"
	"sync"
	"time"
)

type TQ struct {
	// 使用UTC时间；及不要有time.Now().Local()的写法，除非你知道将发生什么

	// 将按照任务预定时间返回对应的Ts.P; 请及时读取, 否则会阻塞以致影响后续任务
	MQ chan interface{}

	addChan        chan Ts             // 任务增加管道
	taskChans      map[int64](chan Ts) // map的Value(管道)存放任务, Key(int64)是任务管道的id
	endTimes       map[int64]time.Time // 记录对应任务管道的最后任务的执行时间
	idsChan        chan int64          // 传递id，表示新建了管道
	lock           sync.Mutex          // 读写锁
	defaultChanLen int                 // 默认任务管道容量
}

// Ts 表示一个任务
type Ts struct {
	T time.Time   // 预定执行UTC时间
	P interface{} // 执行时返回的数据
}

// Run 启动
func (t *TQ) Run() {
	t.defaultChanLen = 64
	t.addChan = make(chan Ts, t.defaultChanLen)
	t.MQ = make(chan interface{}, t.defaultChanLen)
	t.idsChan = make(chan int64, t.defaultChanLen)
	t.taskChans = make(map[int64](chan Ts))
	t.endTimes = make(map[int64]time.Time)

	// 运行新增管道服务
	go func() {
		for {
			// 新建了管道
			id := <-t.idsChan
			go t.exec(t.taskChans[id], id) // 执行每个管道中的任务

			// select {
			// case id := <-t.idsChan:
			// 	go t.exec(t.taskChans[id], id) // 执行每个管道中的任务
			// case <-time.After(time.Minute):
			// 	// nothing
			// }
		}
	}()

	// 分发任务
	go func() {
		var r Ts
		for {
			r = <-t.addChan

			var flag bool = false
			for id, v := range t.endTimes {

				if r.T.After(v) && len(t.taskChans[id]) < t.defaultChanLen { //追加
					t.taskChans[id] <- r
					t.endTimes[id] = r.T
					flag = true
					break
				}
			}
			// 需要新建管道
			if !flag {
				var sc chan Ts = make(chan Ts, t.defaultChanLen*2)
				var id int64
				if len(t.endTimes) == 0 {
					id = 0
				} else {
					id = t.randId()
				}
				t.taskChans[id] = sc // add

				t.endTimes[id] = r.T
				t.taskChans[id] <- r
				t.idsChan <- id
			}

		}
	}()
}

// Add 增加任务
func (t *TQ) Add(r Ts) error {
	if cap(t.addChan)-len(t.addChan) < 1 {
		return errors.New("channel block! len:" + strconv.Itoa(len(t.addChan)) + " ,cap:" + strconv.Itoa(cap(t.addChan)))
	}
	t.addChan <- r
	return nil
}

// exec 执行任务
func (t *TQ) exec(c chan Ts, id int64) {
	var ts Ts
	for {

		t.lock.Lock()
		if id != 0 && len(c) == 0 {
			// 执行完任务后释放任务管道
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

// randId 随机数
func (t *TQ) randId() int64 {
	b := new(big.Int).SetInt64(time.Now().UnixNano())
	i, err := rand.Int(rand.Reader, b)
	if err != nil {
		return 63
	}
	r := i.Int64() + time.Now().UnixNano()
	return r
}
