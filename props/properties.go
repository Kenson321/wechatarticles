package props

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
	"math/rand"
	"wechatarticles/log"
)

const propFileName = "参数.wechat"
const cacheFileName = "缓存.wechat"

func init() {
	initProps()
	initCache()
}

type Properties struct {
	Chrome      string   `json:"chrome"` //本地的chrome执行程序
	WorkDir     string   `json:"wdir"`
	FixDIR      bool     `json:"fdir"`
	JsonFN      string   `json:"jsonfn"`
	TJsonFN     string   `json:"tjsonfn"`
	SupportMail bool     `json:"mail"`
	MailUser    string   `json:"muser"`
	MailPwd     string   `json:"mpwd"`
	MailSubj    string   `json:"msubj"`
	MailDir     string   `json:"mdir"`
	MailTO      []string `json:"mto"`
	MailCC      []string `json:"mcc"`
	MailBCC     []string `json:"mbcc"`
	MailKeys    []string `json:"mkeys"`
	WechatUser  string   `json:"wuser"`
	WechatPwd   string   `json:"wpwd"`
	BeginDay    string   `json:"bday"`
	EdnDay      string   `json:"eday"`
	Image       bool     `json:"image"`
	Sources     []Source `json:"sources"`
}

type Source struct {
	Names              []string `json:"snames"`
	Tag                string   `json:"stag"`
	HighlightMailWords []string `json:"skeys"`
}

var Ppt *Properties

func initProps() {
	Ppt = &Properties{
		Chrome:      `C:\Program Files\Google\Chrome\Application\chrome.exe`,
		WorkDir:     ``,
		FixDIR:      false,
		JsonFN:      `内容.json`,
		TJsonFN:     `内容_tmp.json`,
		SupportMail: true,
		MailUser:    "xxx@mail.com",
		MailPwd:     "password123456",
		MailSubj:    "微信爬虫",
		MailDir:     ``,
		MailTO:      []string{"第一个主送邮箱同步作为扫码授权的主送邮箱@mail.com", "xxx@mail.com"},
		MailCC:      []string{"xxx@mail.com", "xxx@mail.com"},
		MailBCC:     []string{"第一个暗送邮箱同步作为扫码授权的抄送邮箱@mail.com", "xxx@mail.com"},
		MailKeys:    []string{"邮件高亮关键字"},
		WechatUser:  "WechatUserName",
		WechatPwd:   "WechatUserPassword",
		BeginDay:    time.Now().Add(time.Hour * -24).Format("2006-01-02"),
		EdnDay:      time.Now().Add(time.Hour * -24).Format("2006-01-02"),
		Sources: []Source{
			{Names: []string{"aaa"}, Tag: "bbb", HighlightMailWords: []string{"ccc", "ddd"}},
			{Names: []string{"eee"}, Tag: "fff", HighlightMailWords: []string{"ggg", "hhh"}},
		},
	}

	jsonF, err := os.Open(propFileName)
	if err != nil {
		log.Error("打开文件失败", err)
		initPropFile()
		panic(err)
	}
	defer jsonF.Close()

	b, err := ioutil.ReadAll(jsonF)
	if err != nil {
		log.Error("读取文件失败：", err)
		panic(err)
	}

	err = json.Unmarshal(b, &Ppt)
	if err != nil {
		log.Error("解析json失败", err)
		panic(err)
	}

	if len(Ppt.BeginDay) != 10 || len(Ppt.EdnDay) != 10 {
		Ppt.BeginDay = time.Now().Add(time.Hour * -24).Format("2006-01-02")
		Ppt.EdnDay = Ppt.BeginDay
	}

	if len(Ppt.MailSubj) < 1 {
		title1 := []string{"邮件标题前缀", "通过数组支持排列组合避免被误判为垃圾邮件"}
		title2 := []string{"邮件标题后缀", "通过数组支持排列组合避免被误判为垃圾邮件"}
		rand.Seed(time.Now().UnixNano())
		i := rand.Int31() % 7
		j := rand.Int31() % 7
		if strings.EqualFold(Ppt.EdnDay, Ppt.BeginDay) {
			Ppt.MailSubj = fmt.Sprintf("%s%s%s(%s)", title1[i], time.Now().Format("2006-01-02"), title2[j], Ppt.BeginDay)
		} else {
			Ppt.MailSubj = fmt.Sprintf("%s%s%s(%s至%s)", title1[i], time.Now().Format("2006-01-02"), title2[j], Ppt.BeginDay, Ppt.EdnDay)
		}
	}
	
	if !Ppt.FixDIR {
		if strings.EqualFold(Ppt.EdnDay, Ppt.BeginDay) {
			Ppt.WorkDir = filepath.Join(Ppt.WorkDir, fmt.Sprintf("爬虫%s(%s)", time.Now().Format("2006-01-02"), Ppt.BeginDay))
		} else {
			Ppt.WorkDir = filepath.Join(Ppt.WorkDir, fmt.Sprintf("爬虫%s(%s_%s)", time.Now().Format("2006-01-02"), Ppt.BeginDay, Ppt.EdnDay))
		}
	}
}

func initPropFile() {
	jsonF, err := os.Create(propFileName)
	if err != nil {
		log.Error("打开文件失败", err)
		return
	}
	defer jsonF.Close()

	b, _ := json.Marshal(Ppt)
	var bb bytes.Buffer
	json.Indent(&bb, b, "", "\t")

	fmt.Fprintf(jsonF, "%s", bb.String())
}

type Cache struct {
	Cookie  string            `json:"cookie"`
	Token   string            `json:"token"`
	NameFakeIds []NameId          `json:"nameFakeIds"`
	FakeIds map[string]string `json:"-"`
}

type NameId struct {
	Name   string `json:"name"`
	FakeId string `json:"fakeid"`
}

var CachePpt *Cache

func initCache() {
	CachePpt = &Cache{}
	
	CachePpt.FakeIds = make(map[string]string)

	jsonF, err := os.Open(cacheFileName)
	if err != nil {
		log.Error("打开文件失败", err)
		UpdateCacheFile()
		return
	}
	defer jsonF.Close()

	b, err := ioutil.ReadAll(jsonF)
	if err != nil {
		log.Error("读取文件失败：", err)
		return
	}

	err = json.Unmarshal(b, &CachePpt)
	if err != nil {
		log.Error("解析json失败", err)
		return
	}

	for _, nid := range CachePpt.NameFakeIds {
		CachePpt.FakeIds[nid.Name] = nid.FakeId
	}
}

func UpdateCacheFile() {
	jsonF, err := os.Create(cacheFileName)
	if err != nil {
		log.Error("创建文件失败", err)
		return
	}
	defer jsonF.Close()

	b, _ := json.Marshal(CachePpt)
	var bb bytes.Buffer
	json.Indent(&bb, b, "", "\t")
	
	_, err = fmt.Fprintln(jsonF, bb.String())
	if err != nil {
		log.Error("更新缓存文件失败", err)
		return
	}
}
