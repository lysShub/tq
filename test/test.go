package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/lysShub/tq"
)

func main() {

	var q = new(tq.TQ)
	go q.Run()
	// time.Sleep(time.Microsecond * 10)

	// 读取
	q.A = time.Now()
	var r interface{}
	go func() {
		for {
			r = (<-(q.MQ)) //读取任务

			v, ok := r.(string)
			if ok {
				go func() {
					t := time.Now().Sub(q.A)
					fmt.Println(v, t)
				}()

			} else {
				fmt.Println("不是字符串")
			}
		}
	}()

	fmt.Println("写入")

	for i := 1000; i < 1100; i = i + 10 {
		q.Add(tq.Ts{
			T: time.Now().Add(time.Millisecond * time.Duration(i)),
			P: "延时" + strconv.Itoa(i) + "ms",
		})
	}

	q.Add(tq.Ts{
		T: time.Now().Add(time.Second * time.Duration(2)),
		P: "延时 2s",
	})

	time.Sleep(time.Hour)
	fmt.Println("完成")
}
