ksubdomain是一款基于无状态的子域名爆破工具，类似无状态端口扫描，支持在Windows/Linux/Mac上进行快速的DNS爆破，在Mac和Windows上理论最大发包速度在30w/s,linux上为160w/s。

hacking8信息流的src资产收集 https://i.hacking8.com/src/ 用的是ksubdomain

## 优化
在原ksubdomain的代码上进行了优化,精简了一些功能，只专注于快速子域名爆破。因为没有多网卡环境，此版本只支持单网卡。

## 参考
- 原ksubdomain https://github.com/knownsec/ksubdomain
- 从 Masscan, Zmap 源码分析到开发实践 <https://paper.seebug.org/1052/>
- ksubdomain 无状态域名爆破工具介绍 <https://paper.seebug.org/1325/>