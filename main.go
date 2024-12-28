package main

import (
//	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"wechatarticles/chrome"
	"wechatarticles/http"
	"wechatarticles/log"
	"wechatarticles/mail"
	"wechatarticles/props"
)

const logname = `日志.log`

func crawl() {
	jsonF, err := os.Create(filepath.Join(props.Ppt.WorkDir, props.Ppt.JsonFN))
	if err != nil {
		log.Error("打开文件失败", err)
	}
	defer jsonF.Close()

	tmpJF, err := os.Create(filepath.Join(props.Ppt.WorkDir, props.Ppt.TJsonFN))
	if err != nil {
		log.Error("打开文件失败", err)
	}
	defer tmpJF.Close()

	cookie := props.CachePpt.Cookie
	token := props.CachePpt.Token
	fakeid := http.GetFakeid(cookie, token, "逻辑思维")
	if len(fakeid) != len("MjM5NjAxOTU4MA==") {
		log.Info("token, cookie 已过期，将重新获取并更新本地缓存记录")
		token, cookie = chrome.GetAuth()
		props.CachePpt.Cookie = cookie
		props.CachePpt.Token = token
		props.UpdateCacheFile()
	}
	
	updateCache := false
	for _, src := range props.Ppt.Sources {
		for _, name := range src.Names {
			if len(props.CachePpt.FakeIds[name]) < 1 {
				log.Info("新增缓存fakeid记录", name)
				fakeid := http.GetFakeid(cookie, token, name)
				props.CachePpt.FakeIds[name] = fakeid
				props.CachePpt.NameFakeIds = append(props.CachePpt.NameFakeIds, props.NameId{Name: name, FakeId:fakeid})
				updateCache = true
			}
		}
	}
	if updateCache {
		props.UpdateCacheFile()
	}


	log.Info("下载日期：", props.Ppt.BeginDay, props.Ppt.EdnDay)

	articles := make([]http.Article, 0, 200)
	for _, src := range props.Ppt.Sources {
		for _, name := range src.Names {
			arts := http.GetArticleList(cookie, token, props.CachePpt.FakeIds[name], props.Ppt.BeginDay, props.Ppt.EdnDay)
			for _, art := range arts {
				log.Debug("获取文章：", name, art.Title)
	
				art.Source = name
				art.Tag = src.Tag
				//只下载文字送入true，需要下载图片送入false
				art.Content = chrome.Visit(art.Link, !props.Ppt.Image)
				art.Content_hex = base64.StdEncoding.EncodeToString([]byte(art.Content))
	
				articles = append(articles, art)
	
				js, err := json.Marshal(art)
				if err != nil {
					log.Error("转换为json失败", err)
				} else {
					fmt.Fprintln(tmpJF, string(js))
				}
	
				time.Sleep(time.Second * 3)
			}
			time.Sleep(time.Second * 10)
		}
	}
	
	js, err := json.Marshal(articles)
	if err != nil {
		log.Error("转换为json失败", err)
		return
	}
	var bb bytes.Buffer
	json.Indent(&bb, js, "", "\t")
	fmt.Fprintln(jsonF, bb.String())
}

func main() {
	
	dir := props.Ppt.MailDir
	if len(dir) < 1 { //非补发邮件
		dir = props.Ppt.WorkDir
		if props.Ppt.FixDIR {
			os.RemoveAll(dir)
		}
		os.MkdirAll(dir, os.ModeDir|os.ModePerm)

		//日志开关
		log.SetDebug(false, filepath.Join(dir, logname))
	
		crawl()
	}

	if props.Ppt.SupportMail == true {
		mail.SendResult(dir, props.Ppt.JsonFN, props.Ppt.MailUser, props.Ppt.MailPwd, []string{filepath.Join(dir, props.Ppt.JsonFN)})
	}
}
