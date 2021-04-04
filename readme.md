# tq(time queue)：时间任务队列



### 前言：

​		正如你所见，此任务队列使用`channel`实现，设计巧妙，仅使用Golang 的官方包`sync`和`time`实现了所需的功能；优点不言自明。



### 开始

GO111MODULE=on

[参考](https://github.com/lysShub/tq/blob/master/test/test.go)

### 特性

- 轻量
- 并发安全

### 注意

- 精度不高，间隔出现大约`0.01s`的大误差，所以可靠精度`0.1s`，请勿用于高时间精度、低时间延时的业务。但是此误差不累积、延时1s和1h的误差是一样的。理论上不应该有这么大的误差，如果谁有想法请务必告诉我。
- 请用协程运行初始化函数，即`go tq.Run()`；同时使用`tq.InitEnd.Wait()`等待初始化完成。