package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/lysShub/tq"
)

func main() {
	Q := new(tq.TQ)
	go Q.Run()
	tq.InitEnd.Wait() // 必须，确保初始化完成

	var st time.Time

	var r interface{}
	st = time.Now()
	go func() {

		for {
			r = <-(Q.MQ)
			v, ok := r.(string)
			if ok {
				fmt.Println(v, " 实际延时:", time.Now().Sub(st))
			} else {
				fmt.Println("类型错误")
			}
		}
	}()

	for i := 0; i < 20; i++ {
		go Q.Add(tq.Ts{ // 并发安全
			T: time.Now().Add(time.Second * time.Duration(i)),
			P: "设定延时:" + strconv.Itoa(i) + "s",
		})

	}

	time.Sleep(time.Second * 25)

}
