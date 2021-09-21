package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/lysShub/tq"
)

func main() {
	// 测试MQ阻塞，写入也会阻塞

	Q := tq.NewTQ() // 运行

	var st = time.Now()
	go func() {
		return
		var r interface{}
		for r = range Q.MQ {
			// 每次循环执行时间不能太长，避免没有及时读取MQ中数据导致阻塞
			if v, ok := r.(string); ok {
				fmt.Println(v, " 实际延时:", time.Since(st))
				time.Sleep(time.Second)
			}
		}
	}()

	for i := 0; i < 40; i++ {
		Q.Add(tq.Ts{ // 并发安全
			T: time.Now().Add(time.Second).Add(time.Millisecond * time.Duration(i*10)),
			P: "设定延时:" + strconv.Itoa(i) + "s",
		})
		time.Sleep(time.Millisecond * 200)
		fmt.Println(i)
	}

	time.Sleep(time.Hour)
}

func main1() {

	Q := tq.NewTQ() // 运行

	var st = time.Now()
	go func() {
		var r interface{}
		var i int
		for r = range Q.MQ {
			// 每次循环执行时间不能太长，避免没有及时读取MQ中数据导致阻塞
			if v, ok := r.(string); ok {
				fmt.Println(v, " 实际延时:", time.Since(st))
				i++
				if i > 5 {
					return
				}
			}
		}
	}()

	for i := 0; i < 20; i++ {
		go Q.Add(tq.Ts{ // 并发安全
			T: time.Now().Add(time.Second * time.Duration(i)),
			P: "设定延时:" + strconv.Itoa(i) + "s",
		})
	}
	Q.Add(tq.Ts{
		T: time.Now().UTC().Add(time.Second * 10),
		P: "设定延时:10s",
	})

	time.Sleep(time.Second * 21)
	Q.Drop()
}
