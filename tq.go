package tq

import (
	"errors"
	"sync"
	"time"
)

type TQ struct {
	// 时间任务队列
	// 使用slice来存储任务，在中间插入任务会导致slice产生副本；尽量少用
	// 使用UTC时间，及不要有time.Now().Local()的写法；除非你知道将发生什么

	// 将按照预定时间返回消息
	MQ chan interface{}

	/* 内部 */
	container []Ts           // 容器，按照时间顺序存储任务
	wg        sync.WaitGroup // 流程控制阻塞
	mt        sync.RWMutex   // container的写锁
	rest      sync.WaitGroup // 休息阻塞 休息即container中没有数据

}

//
type Ts struct {
	T time.Time
	P interface{} // 任务参数，到时通过MQ返回
}

// Run 运行任务。是阻塞函数、请使用协程运行
func (t *TQ) Run() {
	t.MQ = make(chan interface{}, 64)
	t.container = make([]Ts, 0, 64)

	go func() { // 流程控制

		var inw sync.WaitGroup
		var tmp Ts
		inw.Add(1)
		t.rest.Add(1)

		for {
			t.wg.Add(1)
			go func() {
				go func() {
					for {
						t.rest.Wait() // 休息阻塞

						time.Sleep(t.container[0].T.Sub(time.Now()))

						/* 执行任务 */
						t.mt.RLock()
						tmp = t.container[0]
						t.container = t.container[1:] // 出1
						t.mt.RUnlock()

						t.MQ <- tmp.P // 执行操作

						if len(t.container) == 0 { // 开始休息
							t.rest.Add(1)
						}
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
	if f.T.Before(time.Now()) {
		return errors.New("invalid time")
	}

	if t.insert(f) { // 在最前面插入，须重置延时
		t.wg.Done()
	}
	return nil
}

// 插入任务 返回是否是需要重置延时
func (t *TQ) insert(i Ts) bool {
	var l int = len(t.container)
	var flag bool = false

	if l == 0 { // 取消休息(取消休息不可能需要重置)
		t.mt.RLock()
		t.container = append(t.container, i)
		t.mt.RUnlock()

		// time.Sleep(time.Millisecond * 5)
		// fmt.Println("解除休息阻塞")
		t.rest.Done() // 解除休息阻塞

	} else { // 插入任务

		if i.T.After(t.container[l-1].T) { // 最后追加
			t.mt.RLock()
			t.container = append(t.container, i)
			t.mt.RUnlock()

		} else if i.T.Before(t.container[0].T) { // 开头插入
			t.mt.RLock()
			t.container = append([]Ts{i}, t.container...)
			t.mt.RUnlock()
			flag = true //需要重置

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
			t.mt.RLock()
			// t.container = append(append(t.container[:k+1], i), t.container[k+1:]...)
			t.container = sliceInsert(i, k, t.container)
			t.mt.RUnlock()
		}
	}
	return flag
}

// 在list的下标为的k元素之后插入i；会产生副本
func sliceInsert(i Ts, k int, list []Ts) []Ts {
	var tmp []Ts = make([]Ts, k+1, k+2)
	copy(tmp, list[:k+1])
	tmp = append(tmp, i)
	list = append(tmp, list[k+1:]...)
	return list
}
