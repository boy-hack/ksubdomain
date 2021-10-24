ksubdomain是一款基于无状态的子域名爆破工具，类似无状态端口扫描，支持在Windows/Linux/Mac上进行快速的DNS爆破，在Mac和Windows上理论最大发包速度在30w/s,linux上为160w/s。
## 为什么这么快
ksubdomain使用pcap模块直接从网卡发送数据和接收数据包，即使高并发发包，也不会占用系统描述符让系统网络阻塞。

可以用`--test`来测试本地最大发包数,但实际发包的多少和网络情况息息相关，ksubdomain将网络参数简化为了`-b`参数，输入你的网络下载速度如`-b 5m`，ksubdomain将会自动限制发包速度。
## 可靠性
类似masscan,这么大的发包速度意味着丢包也会非常严重，ksubdomain有丢包重发机制(这样意味着速度会减小，但比普通的DNS爆破快很多)，会保证每个包都收到DNS服务器的回复，漏报的可能性很小。

## 优化
在原ksubdomain的代码上进行了优化,精简了一些功能，只专注于快速子域名爆破。

## 参考
- 从 Masscan, Zmap 源码分析到开发实践 <https://paper.seebug.org/1052/>
- ksubdomain 无状态域名爆破工具介绍 <https://paper.seebug.org/1325/>