package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/lysShub/tq"
)

func main() {
	Q := new(tq.TQ)
	Q.Run() // 运行

	var st = time.Now()
	go func() {
		var r interface{}
		for {
			r = <-(Q.MQ)
			if v, ok := r.(string); ok {
				go fmt.Println(v, " 实际延时:", time.Now().Sub(st))
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
		T: time.Now().UTC().Add(time.Second * 20),
		P: "设定延时: 20s",
	})

	time.Sleep(time.Second * 21)

}
