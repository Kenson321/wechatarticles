package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"
	"wechatarticles/chrome"
	"wechatarticles/http"
	"wechatarticles/log"
)

//本地的chrome执行程序
var bin = `C:\Program Files\Google\Chrome\Application\chrome.exe`

func main() {
	dir := `D:\爬虫` + time.Now().Format("2006-01-02") + `\`
	os.MkdirAll(dir, os.ModeDir|os.ModePerm)

	//日志开关
	log.SetDebug(false, dir+`日志.log`)

	file1, err := os.Create(dir + `内容.txt`)
	if err != nil {
		log.Error("打开文件失败", err)
	}
	defer file1.Close()

	token, cookie := chrome.GetAuth(bin, "微信公众号账号", "微信公众号密码")

	//需要下载其文章的公众号
	//fakeid := http.GetFakeid(cookie, token, "微信公众号名称") //微信公众号ID
	targets := [][]string{
		{"微信公众号ID", "微信公众号名称", "自定义分类标签"},
	}
	
	articles := make([]http.Article, 0, 200)
	for _, target := range targets {
		//下载的日期范围
		arts := http.GetArticleList(cookie, token, target[0], "2024-07-01", "2024-07-10")
		for _, art := range arts {
			fmt.Fprintln(file1, target[1], art.Title, art.Time, art.Class)
			
			art.Source = target[1]
			art.Tag = target[2]
			//只下载文字送入true，需要下载图片送入false
			art.Content = chrome.Visit(bin, dir, art.Link, true)
			art.Content_hex = base64.StdEncoding.EncodeToString([]byte(art.Content))

			articles = append(articles, art)

			fmt.Fprintln(file1, art.Content)
			fmt.Fprintln(file1, art.Link)
			fmt.Fprintln(file1, "")
			fmt.Fprintln(file1, "")
			fmt.Fprintln(file1, "")
			
			time.Sleep(time.Second * 3)
		}
		time.Sleep(time.Second * 10)
	}
}
