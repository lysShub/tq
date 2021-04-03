package tq

import (
	"errors"
	"sync"
	"time"
)

var GuardWg sync.WaitGroup

func init() {
	GuardWg.Add(1)
}

type TQ struct {
	// 时间任务队列
	// 使用slice来存储任务，在中间插入任务会导致slice产生副本；尽量少用
	// 使用UTC时间，及不要有time.Now().Local()的写法；除非你知道将发生什么

	// 将按照预定时间返回消息；请及时读取，否则会阻塞以致影响后续任务
	MQ chan interface{}

	/* 内部 */
	container []Ts           // 容器，按照时间顺序存储任务
	wg        sync.WaitGroup // 流程控制阻塞
	// rwg       sync.WaitGroup // wg的'ACK'，确保在wg的计数器不为负数
	cl   sync.RWMutex   // container的锁
	rest sync.WaitGroup // 休息阻塞；休息即container中没有数据
	// wa        sync.Mutex
	first bool // 懒得初始化，false 表示第一次
	A     time.Time
}

//
type Ts struct {
	T time.Time
	P interface{} // 任务参数，到时通过MQ返回
}

// Run 运行任务。是阻塞函数、请使用协程运行
func (t *TQ) Run() {
	t.container = make([]Ts, 0, 64)
	t.MQ = make(chan interface{}, 64)

	go func() { // 流程控制

		var inw sync.WaitGroup
		var tmp Ts
		// t.rest.Add(1)
		// t.rwg.Add(1)

		// 这种设计失败，并不能是旧协程退出
		for {
			inw.Add(1)
			t.wg.Add(1)
			// t.rwg.Done()

			go func() {
				go func() {
					for {

						if len(t.container) == 0 { // 休息阻塞
							t.rest.Add(1)
							if !t.first {
								GuardWg.Done()
								t.first = true
							}
						}

						// a := time.Now()
						t.rest.Wait() // 休息阻塞

						time.Sleep(t.container[0].T.Sub(time.Now()))

						tmp = t.container[0]

						/* 执行任务 */
						t.cl.Lock()

						t.container = t.container[1:] // 出1
						t.cl.Unlock()

						t.MQ <- tmp.P // 执行操作

					}
				}()
				t.wg.Wait()

				inw.Done()
			}()
			inw.Wait()
		}
	}()
}

// Add 增加一个任务
func (t *TQ) Add(f Ts) error {

	t.cl.Lock()

	if f.T.Before(time.Now()) {
		t.cl.Unlock()
		return errors.New("invalid time")
	}

	r := t.insert(f)

	t.cl.Unlock()
	if r == 2 { // 在最前面插入，须重置延时
		t.wg.Done()

	} else if r == 1 {
		if !t.first {
			GuardWg.Wait()
		}
		t.rest.Done()
		if t.first {

		}
	}

	return nil
}

// TaskLoad 当前任务量
func (t *TQ) TaskLoad() int {
	r := len(t.container)
	return r
}

/*
* 私有
 */

// 插入任务 返回是否是需要重置延时
func (t *TQ) insert(i Ts) int {

	var l int = len(t.container)
	var flag int = 0

	if l == 0 { // 取消休息(取消休息不可能需要重置)
		t.container = append(t.container, i)

		flag = 1

	} else { // 插入任务

		if i.T.After(t.read(l - 1).T) { // 最后追加
			t.container = append(t.container, i)

		} else if i.T.Before(t.read(0).T) { // 开头插入
			t.container = append([]Ts{i}, t.container...)
			flag = 2 //需要重置

		} else { // 其他位置插入
			k := func() int { // 二分法 查找
				left, right, mid := 0, l-1, 0
				for {
					mid = (left + right) / 2
					if i.T.Before(t.container[mid].T) { // 太大
						right = mid
					} else if i.T.After(t.container[mid].T) { // 太小
						left = mid
					} else { //相等
						return mid
					}
					if right-left <= 2 {
						for j := left; j <= right; j++ {
							if i.T.Before(t.container[j].T) {
								return j - 1
							}
						}
					}
				}
			}()

			// 插入
			t.container = sliceInsert(i, k, t.container)
		}
	}
	return flag
}

func (t *TQ) read(l int) Ts {
	return t.container[l]
}

// 在list的下标为的k元素之后插入i；会产生副本
func sliceInsert(i Ts, k int, list []Ts) []Ts {
	var tmp []Ts = make([]Ts, k+1, k+2)
	copy(tmp, list[:k+1])
	tmp = append(tmp, i)
	list = append(tmp, list[k+1:]...)
	return list
}
