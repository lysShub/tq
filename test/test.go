package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/lysShub/tq"
)

func main() {

	var q = new(tq.TQ)
	q.InitEnd.Add(1)
	go q.Run()
	q.InitEnd.Wait() // 确保初始化完成

	// 读取
	star := time.Now()
	var r interface{}
	go func() {
		for {
			r = (<-(q.MQ)) //读取任务

			v, ok := r.(string)
			if ok {
				go func() {
					t := time.Now().Sub(star)
					fmt.Println(v, "   实际延时:", t)
				}()

			} else {
				fmt.Println("不是字符串")
			}
		}
	}()

	// 写入任务
	for i := 1; i < 100; i++ {
		go q.Add(tq.Ts{ // 并发安全
			T: time.Now().Add(time.Second * time.Duration(i)),
			P: "延时" + strconv.Itoa(i) + "s",
		})
	}

	time.Sleep(time.Hour)
}
