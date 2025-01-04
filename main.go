package main

import (
//	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		panic(err)
	}
	defer jsonF.Close()
	
	var txtF *os.File = nil
	if len(props.Ppt.TxtFN) > 0 {
		txtF, err = os.Create(filepath.Join(props.Ppt.WorkDir, props.Ppt.TxtFN))
		if err != nil {
			panic(err)
		}
		defer jsonF.Close()
	}

	tmpJF, err := os.Create(filepath.Join(props.Ppt.WorkDir, props.Ppt.TJsonFN))
	if err != nil {
		panic(err)
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


	log.Info("爬取信息日期范围：", props.Ppt.BeginDay, props.Ppt.EdnDay)

	articles := make([]http.Article, 0, 200)
	for _, src := range props.Ppt.Sources {
		log.Debug("爬取公众号类型：", src.Tag)
		for _, name := range src.Names {
			log.Debug("爬取公众号：", name)
			arts := http.GetArticleList(cookie, token, props.CachePpt.FakeIds[name], props.Ppt.BeginDay, props.Ppt.EdnDay)
			time.Sleep(time.Second * 10)
			log.Debug("公众号文章数量：", name, len(arts))
			for _, art := range arts {
				log.Debug("爬取文章：", name, art.Title)
	
				art.Source = name
				art.Tag = src.Tag
				art.Content = chrome.Visit(art.Link)
				time.Sleep(time.Second * 3)
				art.Content_hex = base64.StdEncoding.EncodeToString([]byte(art.Content))
				
				if src.MustMatch { //标题和内容必须包含关键字才需要记录
					match := false
					for _, kw := range src.HighlightMailWords {
						if strings.Contains(art.Title, kw) || strings.Contains(art.Content, kw) {
							match = true
							break
						}
					}
					if !match {
						continue
					}
				}
	
				//汇总结果
				articles = append(articles, art)
	
				//写临时文件
				js, err := json.Marshal(art)
				if err != nil {
					log.Error("转换为json失败", err)
				} else {
					fmt.Fprintln(tmpJF, string(js))
				}
				
				//写txt文件
				if txtF != nil {
					fmt.Fprintln(txtF, art.Source, art.Title, art.Time)
					fmt.Fprintln(txtF, art.Content)
					fmt.Fprintln(txtF, "")
				}
			}
		}
	}
	
	//写汇总文件
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

	if props.Ppt.OnlyMail == false { //非补发邮件
		dir := props.Ppt.WorkDir
		os.RemoveAll(dir)
		os.MkdirAll(dir, os.ModeDir|os.ModePerm)

		//日志开关
		log.SetDebug(props.Ppt.Debug, filepath.Join(dir, logname))
	
		crawl()
	}

	if props.Ppt.SupportMail == true {
		mail.SendResult()
	}
}
