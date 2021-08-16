package tq

import (
	"sync"
	"time"
)

type TQ struct {

	// 达到任务执行时间时返回对应的Ts.P; 请及时读取, 否则会阻塞以致影响后续任务
	MQ chan interface{}
	// 赋值True将销毁队列实例; MQ将不在有返回, 处理work的协程将退出, Add将会painc
	Close bool

	addChan chan Ts     // 增加任务管道
	works   []*work     // 记录任务
	lock    sync.Mutex  // 互斥锁, 确保
	tricker time.Ticker // 周期通知
}

// Ts 表示一个任务
type Ts struct {
	T time.Time   // 设定执行时间
	P interface{} // 执行时MQ返回的数据
}

// work 表示一个工作
type work struct {
	c       chan Ts
	endTime time.Time
}

const workLen = 1024 * 16 // work.c管道容量

func NewTQ() *TQ {
	var t = new(TQ)
	t.run()
	return t
}

// Run 启动
func (t *TQ) run() {
	t.MQ = make(chan interface{}, 128)
	t.Close = false
	t.addChan = make(chan Ts, 128)
	t.works = make([]*work, 0, 16)
	t.tricker = *time.NewTicker(time.Second * 5)

	// 分发任务
	go func() {
		var r Ts
		var flag bool

		for !t.Close {

			select {
			case r = <-t.addChan:
				flag = false
				for i := 0; i < len(t.works); i++ {
					if r.T.After(t.works[i].endTime) && len(t.works[i].c) < workLen {
						t.works[i].endTime = r.T
						t.works[i].c <- r
						flag = true
						break
					}
				}

				// 需要新建work
				if !flag {
					var tc chan Ts = make(chan Ts, workLen)

					var w = new(work)
					w.c = tc
					w.endTime = r.T

					t.lock.Lock()
					t.works = append(t.works, w)
					t.lock.Unlock()

					tc <- r
					// 运行work
					go t.exec(w)
				}
			case <-t.tricker.C:
				//
			}

		}
		close(t.addChan)
	}()
}

// Add 增加任务
func (t *TQ) Add(r Ts) error {
	t.addChan <- r
	return nil
}

// Drop 结束任务队列
func (t *TQ) Drop() {
	t.Close = true
}

// exec 执行work
func (t *TQ) exec(w *work) {
	var ts Ts
	for !t.Close {

		select {
		case ts = <-w.c:
			time.Sleep(time.Until(ts.T)) //延时
			if !t.Close {
				t.MQ <- ts.P
			}
		case <-t.tricker.C:
			// 自动关闭, 结束协程(挂起1个work,避免频繁创建work)
			if len(w.c) == 0 && len(t.works) > 1 {
				t.lock.Lock()
				for i := 0; i < len(t.works); i++ {
					if t.works[i] == w {
						t.works = append(t.works[:i], t.works[i+1:]...)
					}
				}
				t.lock.Unlock()
				return
			}
		}

	}
}
