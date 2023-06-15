# hostsdns

一个用go语言实现的类似/etc/hosts功能的本地dns服务器，使用hostsdns可以在支持本地解析的功能上也避免了/etc/hosts被人篡改导致解析失败。

读取指定文件(`/etc/hostsdns`, 文件格式为/etc/hosts格式)，提供离线的dns域名解析服务器。

支持IPV4与IPV6

```shell
[root@ahanwhite dns]# hostsdns --help
hostsdns is a dns server that implements local resolution functions similar to /etc/hosts. 
Using hostsdns can support the local resolution function and avoid the resolution failure caused by tampering of /etc/hosts

Usage:
  hostsdns [flags]

Flags:
  -h, --help              help for hostsdns
      --hostfile string   hostfile for dnsserver (default is /etc/hostdns) (default "/etc/hostdns")
  -p, --port int          port for dns server using udp protocol (default is 53) (default 53)
```

> 参考：  https://juejin.cn/post/6995465840732684295