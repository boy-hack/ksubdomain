ksubdomain是一款基于无状态的子域名爆破工具，类似无状态端口扫描，支持在Windows/Linux/Mac上进行快速的DNS爆破，在Mac和Windows上理论最大发包速度在30w/s,linux上为160w/s。

hacking8信息流的src资产收集 https://i.hacking8.com/src/ 用的是ksubdomain

![](image.gif)

## Useage
```bash
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
提供完整的域名列表，ksubdomain负责快速获取结果 
```bash
./ksubdomain verify -h

NAME:
   cmd verify - 验证模式

USAGE:
   cmd verify [command options] [arguments...]

OPTIONS:
   --filename value, -f value   验证域名文件路径
   --band value, -b value       宽带的下行速度，可以5M,5K,5G (default: "2m")
   --resolvers value, -r value  dns服务器文件路径，一行一个dns地址
   --output value, -o value     输出文件名
   --silent                     使用后屏幕将仅输出域名 (default: false)
   --retry value                重试次数,当为-1时将一直重试 (default: 3)
   --timeout value              超时时间 (default: 6)
   --stdin                      接受stdin输入 (default: false)
   --only-domain, --od          只打印域名，不显示ip (default: false)
   --not-print, --np            不打印域名结果 (default: false)
   --help, -h                   show help (default: false)
```

```
从文件读取 
./ksubdomain v -f dict.txt

从stdin读取
echo "www.hacking8.com"|./ksubdomain v --stdin
```
**枚举模式**
只提供一级域名，指定域名字典或使用ksubdomain内置字典，枚举所有二级域名
```bash
./ksubdomain enum -h

NAME:
   cmd enum - 枚举域名

USAGE:
   cmd enum [command options] [arguments...]

OPTIONS:
   --band value, -b value          宽带的下行速度，可以5M,5K,5G (default: "2m")
   --resolvers value, -r value     dns服务器文件路径，一行一个dns地址
   --output value, -o value        输出文件名
   --silent                        使用后屏幕将仅输出域名 (default: false)
   --retry value                   重试次数,当为-1时将一直重试 (default: 3)
   --timeout value                 超时时间 (default: 6)
   --stdin                         接受stdin输入 (default: false)
   --only-domain, --od             只打印域名，不显示ip (default: false)
   --not-print, --np               不打印域名结果 (default: false)
   --domain value, -d value        爆破的域名
   --domainList value, --dl value  从文件中指定域名
   --filename value, -f value      字典路径
   --skip-wild                     跳过泛解析域名 (default: false)
   --level value, -l value         枚举几级域名，默认为2，二级域名 (default: 2)
   --level-dict value, --ld value  枚举多级域名的字典文件，当level大于2时候使用，不填则会默认
   --help, -h                      show help (default: false)
```

```
./ksubdomain e -d baidu.com

从stdin获取
echo "baidu.com"|./ksubdomain e --stdin
```


## 参考
- 原ksubdomain https://github.com/knownsec/ksubdomain
- 从 Masscan, Zmap 源码分析到开发实践 <https://paper.seebug.org/1052/>
- ksubdomain 无状态域名爆破工具介绍 <https://paper.seebug.org/1325/>
- [ksubdomain与massdns的对比](https://mp.weixin.qq.com/s?__biz=MzU2NzcwNTY3Mg==&mid=2247484471&idx=1&sn=322d5db2d11363cd2392d7bd29c679f1&chksm=fc986d10cbefe406f4bda22f62a16f08c71f31c241024fc82ecbb8e41c9c7188cfbd71276b81&token=76024279&lang=zh_CN#rd) 