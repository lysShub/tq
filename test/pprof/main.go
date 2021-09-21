package main

import (
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/lysShub/tq"
)

func main() {

	go func() {
		test()
	}()

	http.ListenAndServe(":8792", nil)

}

func test() {
	Q := tq.NewTQ() // 运行
	defer Q.Drop()

	go func() {
		var r interface{}
		for r = range Q.MQ {
			// 每次循环执行时间不能太长，避免没有及时读取MQ中数据导致阻塞
			if v, ok := r.(string); !ok {
				panic(v)
			}
		}
	}()

	for i := 0; i < 100000; i++ {
		Q.Add(tq.Ts{
			T: time.Now().Add(time.Millisecond * time.Duration((i % 10))),
			P: "设定延时:" + "s",
		})
		time.Sleep(time.Millisecond)
	}

	time.Sleep(time.Second * 40)

}
