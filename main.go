package main 

import (
	"fmt"
	"os"
	"time"
	"wechatarticles/http"
	"wechatarticles/chrome"
	"wechatarticles/log"
)

//本地的chrome执行程序
var bin = `C:\Program Files\Google\Chrome\Application\chrome.exe`

func main() {
	//日志开关
	log.SetDebug(false)
	
	//保存输出结果的文件
	file1, err := os.Create(`D:\微信.txt`)
	if err != nil {
		log.Println("打开文件失败", err)
	}
	defer file1.Close()

	token, cookie := chrome.GetAuth(bin, "微信公众号账号", "微信公众号密码")

	//需要下载其文章的公众号
	fakeids := []string{
		"微信公众号ID",
	}
	sources := []string{
		"微信公众号名称",
	}
	
	i := 0
	for _, fakeid := range fakeids {
		//下载的日期范围
		arts := http.GetArticleList(cookie, token, fakeid, "2024-07-01", "2024-07-10")
		for _, art := range arts {
			fmt.Fprintln(file1, sources[i], art.Title, art.Time, art.Class, art.Digest)
			
			//只下载文字送入true，需要下载图片送入false
			art.Content = chrome.Visit(bin, `D:\微信\`, art.Link, true)
			time.Sleep(time.Second * 1)
		}
		i++
		time.Sleep(time.Second * 10)
	}
}
