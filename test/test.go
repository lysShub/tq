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

	// 读取
	a := time.Now()
	var r interface{}
	go func() {
		for {
			r = (<-(q.MQ)) //读取任务
			v, ok := r.(string)
			if ok {
				go func() {
					t := time.Now().Sub(a)
					fmt.Println(v, t)
				}()

			} else {
				fmt.Println("不是字符串")
			}
		}
	}()

	fmt.Println("写入")

	// 增加任务
	q.Add(tq.Ts{
		T: time.Now().Add(time.Second),
		P: "延时1s",
	})

	q.Add(tq.Ts{
		T: time.Now().Add(time.Second * 2),
		P: "延时2s",
	})

	q.Add(tq.Ts{
		T: time.Now().Add(time.Second * 3),
		P: "延时3s",
	})

	q.Add(tq.Ts{
		T: time.Now().Add(time.Millisecond * 2500),
		P: "延时2.5s",
	})
	for i := 10; i < 100; i++ {
		q.Add(tq.Ts{
			T: time.Now().Add(time.Second * time.Duration(i)),
			P: "延时" + strconv.Itoa(i) + "s",
		})
	}

	time.Sleep(time.Minute * 5)
}
