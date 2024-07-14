package http

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	"wechatarticles/log"
)

type ArticleList struct {
	Resp struct {
		Msg  string `json:"err_msg"`
		code int    `json:"ret"`
	} `json:"base_resp"`
	Page struct {
		List []struct {
			Info struct {
				Msg []struct {
					Title  string `json:"title"`
					Link   string `json:"link"`
					Time   int64  `json:"update_time"`
					Author string `json:"author_name"`
					Digest string `json:"digest"`
					Class  []struct {
						Title string `json:"title"`
					} `json:"appmsg_album_infos"`
				} `json:"appmsgex"`
			} `json:"publish_info"`
		} `json:"publish_list"`
	} `json:"publish_page"`
}

type Article struct {
	Title  string
	Link   string
	Time   string
	Author string
	Digest string
	Class  string
	Content  string
}

func GetArticleList(cookie, token, fakeid, begData, endDate string) []Article {
	tb, _ := time.Parse("2006-01-02 15:04:05.000", begData+" 00:00:01.000")
	te, _ := time.Parse("2006-01-02 15:04:05.000", endDate+" 23:59:59.000")
	
	arts := make([]Article, 0, 25)
	count := 0
	br := false
	for true {
		begin := fmt.Sprintf("%d", count)
		as := getArticleList(cookie, token, fakeid, begin)
		for _, art := range as {
			t1, _ := time.Parse("2006-01-02 15:04:05.000", art.Time)
			if tb.Sub(t1)>0 {
				br = true
				break
			}
			if (te.Sub(t1)>0) {
				arts = append(arts, art)
			}
		}
		if br {
			break
		}
		count += 5
	}
	return arts
}

func getArticleList(cookie, token, fakeid, begin string) []Article {

	reqUrl := `https://mp.weixin.qq.com/cgi-bin/appmsgpublish`
	agent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:126.0) Gecko/20100101 Firefox/126.0"

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	params := url.Values{}
	params.Set("sub", "list")
	params.Set("search_field", "null")
	params.Set("begin", begin)
	params.Set("count", "5") //5条推送，每条推送可以包含多篇文章
	params.Set("query", "")
	params.Set("fakeid", fakeid)
	params.Set("type", "101_1")
	params.Set("free_publish_type", "1")
	params.Set("sub_action", "list_ex")
	params.Set("token", token)
	params.Set("lang", "zh_CN")
	params.Set("f", "json")
	params.Set("ajax", "1")

	rawUrl, err := url.Parse(reqUrl)
	if err != nil {
		return nil
	}
	rawUrl.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", rawUrl.String(), nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("User-Agent", agent)
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Referer", "https://mp.weixin.qq.com/cgi-bin/appmsgtemplate?action=edit&lang=zh_CN&token="+token)
	req.Header.Add("Cookie", cookie)
	req.Header.Add("Sec-Fetch-Dest", "empty")
	req.Header.Add("Sec-Fetch-Mode", "cors")
	req.Header.Add("Sec-Fetch-Site", "same-origin")
	req.Header.Add("Priority", "u=1")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("发送https请求失败：", err)
		return nil
	}
	defer resp.Body.Close()

	bb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("读取返回失败：", err)
		return nil
	}
	res := string(bb)
	log.Debug("返回：", res)
	res = strings.ReplaceAll(res, `\\`, "")
	res = strings.ReplaceAll(res, `\"`, `"`)
	res = strings.ReplaceAll(res, `"{`, `{`)
	res = strings.ReplaceAll(res, `}"`, `}`)
	log.Println("返回：", res)

	var artl ArticleList
	err = json.Unmarshal([]byte(res), &artl)
	if err != nil {
		log.Println("json解释失败：", err)
		return nil
	}

	log.Println("解释json：", artl.Resp.Msg)

	arts := make([]Article, 0, 25)
	for _, list := range artl.Page.List {
		for _, msg := range list.Info.Msg {
			log.Println("标题：", msg.Title)
			log.Println("地址：", msg.Link)
			log.Println("时间：", msg.Time, time.Unix(int64(msg.Time), 0).Format("2006-01-02 15:04:05.000"))
			log.Println("作者：", msg.Author)

			var art Article
			art.Title = msg.Title
			art.Link = msg.Link
			art.Time = time.Unix(int64(msg.Time), 0).Format("2006-01-02 15:04:05.000")
			art.Author = msg.Author
			art.Digest = msg.Digest
			for _, class := range msg.Class {
				art.Class += class.Title + ";"
			}
			arts = append(arts, art)
		}
	}

	return arts
}
