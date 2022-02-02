ksubdomain是一款基于无状态的子域名爆破工具，类似无状态端口扫描，支持在Windows/Linux/Mac上进行快速的DNS爆破，在Mac和Windows上理论最大发包速度在30w/s,linux上为160w/s。

hacking8信息流的src资产收集 https://i.hacking8.com/src/ 用的是ksubdomain

## Useage
```bash
w8ay@MacBook-Pro  ~/programs/ksubdomain   main ●  ./ksubdomain

NAME:
   KSubdomain - 无状态子域名爆破工具

USAGE:
   ksubdomain [global options] command [command options] [arguments...]

VERSION:
   1.4

COMMANDS:
   enum, e    枚举域名
   verify, v  验证模式
   test       测试本地网卡的最大发送速度
   help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help (default: false)
   --version, -v  print the version (default: false)

```

### 模式
**验证模式**

提供完整的域名列表，ksubdomain负责快速从dns服务器获取结果 
```
./ksubdomain v -f dict.txt
或
echo "www.hacking8.com"|./ksubdomain v --stdin
```
**枚举模式**

只提供一级域名，指定域名字典或使用ksubdomain内置字典，枚举所有二级域名
```
./ksubdomain e -f dict.txt
或
echo "baidu.com"|./ksubdomain e --stdin
```


## 参考
- 原ksubdomain https://github.com/knownsec/ksubdomain
- 从 Masscan, Zmap 源码分析到开发实践 <https://paper.seebug.org/1052/>
- ksubdomain 无状态域名爆破工具介绍 <https://paper.seebug.org/1325/>