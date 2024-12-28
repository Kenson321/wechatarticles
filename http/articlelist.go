package http

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	//	"strings"
	"time"
	"wechatarticles/log"
)

type fakeidResp struct {
	Resp struct {
		Msg  string `json:"err_msg"`
		code int    `json:"ret"`
	} `json:"base_resp"`
	List []struct {
		Fakeid string `json:"fakeid"`
	} `json:"list"`
}

//获取公众号source对应的id
func GetFakeid(cookie, token, source string) (fakeid string) {
	reqUrl := `https://mp.weixin.qq.com/cgi-bin/searchbiz`
	agent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:126.0) Gecko/20100101 Firefox/126.0"

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	params := url.Values{}
	params.Set("action", "search_biz")
	params.Set("begin", "0")
	params.Set("count", "5")
	params.Set("query", source)
	params.Set("token", token)
	params.Set("lang", "zh_CN")
	params.Set("f", "json")
	params.Set("ajax", "1")

	rawUrl, err := url.Parse(reqUrl)
	if err != nil {
		panic(err)
	}
	rawUrl.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", rawUrl.String(), nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("User-Agent", agent)
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	//	req.Header.Add("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("Connection", "keep-alive")
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
	req.Header.Add("Referer", "https://mp.weixin.qq.com/cgi-bin/appmsg?t=media/appmsg_edit_v2&action=edit&isNew=1&type=77&share=1&token="+token+"&lang=zh_CN&timestamp="+timestamp)
	req.Header.Add("Cookie", cookie)
	req.Header.Add("Sec-Fetch-Dest", "empty")
	req.Header.Add("Sec-Fetch-Mode", "cors")
	req.Header.Add("Sec-Fetch-Site", "same-origin")
	req.Header.Add("Priority", "u=0")

	resp, err := client.Do(req)
	if err != nil {
		log.Error("发送https请求失败：", err)
		return
	}
	defer resp.Body.Close()

	bb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("读取返回失败：", err)
		return
	}
	res := string(bb)
	log.Info("返回：", res)

	var fidResp fakeidResp
	err = json.Unmarshal(bb, &fidResp)
	if err != nil {
		log.Error("json解释失败：", err)
		return
	}

	for _, fid := range fidResp.List {
		fakeid = fid.Fakeid
		return
	}

	return
}

type Response struct {
	Resp struct {
		Msg  string `json:"err_msg"`
		code int    `json:"ret"`
	} `json:"base_resp"`
	Page string `json:"publish_page"`
}

type ResponsePage struct {
	List []struct {
		Info string `json:"publish_info"`
	} `json:"publish_list"`
}

type ResponseInfo struct {
	Sinfo struct {
		Time int64 `json:"time"`
	} `json:"sent_info"`
	Msg []struct { //每条推送可以包含多篇文章（消息）
		Title  string `json:"title"`
		Link   string `json:"link"`
		Time   int64  `json:"update_time"` //update_time、create_time
		Author string `json:"author_name"`
		Digest string `json:"digest"`
		Class  []struct {
			Title string `json:"title"`
		} `json:"appmsg_album_infos"`
	} `json:"appmsgex"`
}

type Article struct {
	Source      string `json:"source"`
	Tag         string `json:"tag"`
	Title       string `json:"title"`
	Link        string `json:"link"`
	Time        string `json:"time"`
	Ptime       string `json:"-"`
	Author      string `json:"author"`
	Digest      string `json:"digest"`
	Class       string `json:"class"`
	Content     string `json:"-"`
	Content_hex string `json:"content"`
}

//获取公众号fakeid在begDay和endDay日期范围内的文章列表
func GetArticleList(cookie, token, fakeid, begDay, endDay string) []Article {
	log.Info("公众号：", fakeid)

	tb, _ := time.Parse("2006-01-02 15:04:05.000", begDay+" 00:00:01.000")
	te, _ := time.Parse("2006-01-02 15:04:05.000", endDay+" 23:59:59.000")

	arts := make([]Article, 0, 25)
	count := 0
	br := false
	for true {
		begin := fmt.Sprintf("%d", count)
		as := getArticleList(cookie, token, fakeid, begin)
		if as == nil {
			break
		}
		for _, art := range as {
			//			t1, _ := time.Parse("2006-01-02 15:04:05.000", art.Ptime) //出现了无效日期：1970-01-01
			t1, _ := time.Parse("2006-01-02 15:04:05.000", art.Time)
			if tb.Sub(t1) > 0 {
				br = true
				break
			}
			if te.Sub(t1) > 0 {
				arts = append(arts, art)
			}
		}
		if br {
			break
		}
		count += 5
		time.Sleep(time.Second * 3)
	}
	return arts
}

//获取公众号fakeid的文章列表，一次5条推送，由begin指定起始序号，送0表示由第一条开始
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
	params.Set("count", "5") //5条推送，每条推送可以包含多篇文章（消息）
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
		panic(err)
	}
	rawUrl.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", rawUrl.String(), nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("User-Agent", agent)
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	//	req.Header.Add("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Referer", "https://mp.weixin.qq.com/cgi-bin/appmsgtemplate?action=edit&lang=zh_CN&token="+token)
	req.Header.Add("Cookie", cookie)
	req.Header.Add("Sec-Fetch-Dest", "empty")
	req.Header.Add("Sec-Fetch-Mode", "cors")
	req.Header.Add("Sec-Fetch-Site", "same-origin")
	req.Header.Add("Priority", "u=1")
	//	req.Header.Add("TE", "trailers")

	resp, err := client.Do(req)
	if err != nil {
		log.Error("发送https请求失败：", err)
		return nil
	}
	defer resp.Body.Close()

	bb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("读取返回失败：", err)
		return nil
	}
	res := string(bb)
	log.Info("返回：", res)

	var arResp Response
	err = json.Unmarshal([]byte(res), &arResp)
	if err != nil {
		log.Error("json解释失败：", err)
		return nil
	}
	log.Info("状态：", arResp.Resp.Msg)

	var artPg ResponsePage
	err = json.Unmarshal([]byte(arResp.Page), &artPg)
	if err != nil {
		log.Error("json解释失败：", arResp.Page, err)
		return nil
	}

	arts := make([]Article, 0, 25)
	for _, ap := range artPg.List {
		var artInfo ResponseInfo
		err = json.Unmarshal([]byte(ap.Info), &artInfo)
		if err != nil {
			log.Error("json解释失败：", ap.Info, err)
			return nil
		}

		for _, msg := range artInfo.Msg {
			log.Info("标题：", msg.Title)
			log.Info("地址：", msg.Link)
			log.Info("时间：", msg.Time, time.Unix(int64(msg.Time), 0).Format("2006-01-02 15:04:05.000"))
			log.Info("作者：", msg.Author)

			var art Article
			art.Title = msg.Title
			art.Link = msg.Link
			art.Time = time.Unix(int64(msg.Time), 0).Format("2006-01-02 15:04:05.000")
			art.Ptime = time.Unix(int64(artInfo.Sinfo.Time), 0).Format("2006-01-02 15:04:05.000")
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
