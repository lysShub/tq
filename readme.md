# tq(time queue)：时间任务队列

​			如你所见，此任务队列使用`channel`实现，巧妙设计，仅使用Golang 官方包`sync`和`time`实现了所有功能；优点不言自明。本队列**<font style="color:red">误差不会累积、且并发安全</font>**。

## 快速开始

​		[参考](https://github.com/lysShub/tq/blob/master/test/test.go)

## 工作原理

​		几乎在同一时间依次`Add` `time.Duration`为`1s 2s 3s 8s 12s 5s 7s 4s `的任务，那么[taskChans](https://github.com/lysShub/tq/blob/master/tq.go#L19)将有3个管道用来存储任务：

```shell
id: 0                              任务: 1s 2s 3s 8s 12s
```

```shell
id: 1885497861170684063(随机)       任务：5s 7s
```

```shell
id: 1703901482988809310(随机)       任务：4s
```

​		所以，顺序添加任务是最优的，逆序添加任务是最差的，等效于使用time.After。

## 注意

​		存在误差，误差大小与系统相关；由time.Sleep()的延时的误差导致，测试发现time.Sleep()实际延时时长总是大于设定时长，因此任务总是稍有延迟执行。

```go
var a []time.Duration = make([]time.Duration, 0, 20)
for i := 0; i < 20; i++ {
	s := time.Now()
	time.Sleep(time.Nanosecond)

	a = append(a, time.Since(s))
    // time.Sleep(time.Millisecond * 200)
}
var t time.Duration
for _, v := range a {
	fmt.Println(v)
	t = t + v
}
fmt.Println("平均误差：", t/20)
```

​	系统上的误差可以可以通过以上代码大致了解，在Windows系统上误差存在波动，平均误差甚至可以达到10ms级别（似乎和CPU频率有关，低压U误差更大）；而在Linux上则较为理想。

