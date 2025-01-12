# wechatarticles
batch download articles in wechat, similar to crawer or spider.
批量下载指定的微信公众号推送的所有文章，与爬虫的效果有点类似。

# 版本说明
V0.0.1、V0.0.2 属于学习版本，代码量少，具有较多手工操作步骤
V0.0.3 属于稳定的工具版本，可以作为后台定时任务执行

# 原理介绍
参考：https://kenson321.github.io/2024/06/23/WeChatSpidier/#more

# 使用例子
wechatarticles/main.go

# 执行方式
修改"wechat.properties"文件中的用户名密码以及公众号列表等参数，然后执行如下命令（windows为例）
```
go build
./wechatarticles.exe
```