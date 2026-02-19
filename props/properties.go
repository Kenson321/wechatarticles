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
	"wechatarticles/log"
)

const propFileName = "wechat.properties"
const cacheFileName = "wechat.cache"

func init() {
	initProps()
	initCache()
}

type Properties struct {
	Chrome      string   `json:"chrome"`     //本地的chrome执行程序
	PdfChinese  string   `json:"pdfchinese"` //pdf中文字体路径
	Debug       bool     `json:"debug"`      //输出debug日志
	LogFN       string   `json:"logfn"`      //日志文件
	BaseDir     string   `json:"bdir"`       //主目录
	WorkDir     string   `json:"wdir"`       //工作目录，不指定时支持在主目录下动态创建，指定时则无需指定主目录
	JsonFN      string   `json:"jsonfn"`     //爬虫结果文件，以json文件的形式保存
	TJsonFN     string   `json:"tjsonfn"`    //爬虫结果临时文件，以json文件的形式保存
	TxtFN       string   `json:"txtfn"`      //爬虫结果文件（可选），以txt文件的形式保存
	ExcelFN     string   `json:"excelfn"`    //爬虫结果文件（可选），以excel文件的形式保存
	PdfFN       string   `json:"pdffn"`      //爬虫结果文件（可选），以excel文件的形式保存
	SupportMail bool     `json:"mail"`       //是否支持发送邮件，包括发送二维码扫描邮件以及爬取结果邮件
	OnlyMail    bool     `json:"onlymail"`   //只发邮件，已完成爬取发送邮件失败需要重发时，配合WorkDir使用
	MailUser    string   `json:"muser"`      //发送邮件的邮箱用户名
	MailPwd     string   `json:"mpwd"`       //发送邮件的邮箱用户密码
	MailSubj    string   `json:"msubj"`      //发送爬取结果邮件的主题，不指定时动态创建避免误中垃圾邮件判断
	MailAuthTO  []string `json:"mauthto"`    //扫码授权的主送邮箱
	MailTO      []string `json:"mto"`        //主送邮箱
	MailPdfTO   []string `json:"mpdfto"`     //主送邮箱
	MailCC      []string `json:"mcc"`        //抄送邮箱
	MailBCC     []string `json:"mbcc"`       //暗送邮箱
	MailKeys    []string `json:"mkeys"`      //邮件高亮关键字
	WechatUser  string   `json:"wuser"`      //微信公众号用户名
	WechatPwd   string   `json:"wpwd"`       //微信公众号用户密码
	BeginDay    string   `json:"bday"`       //爬取开始日期，不设置默认为上一日
	EdnDay      string   `json:"eday"`       //爬取结束日期，不设置默认为上一日
	Image       bool     `json:"image"`      //下载图片，默认为否，不下载
	Sources     []Source `json:"sources"`    //公众号分类
}

type Source struct {
	Names     []string `json:"snames"` //公众号分类的公众号列表
	Tag       string   `json:"stag"`   //公众号分类
	MustKeys  []string `json:"skeys"`  //公众号分类的关键字
	MustMatch bool     `json:"smatch"` //公众号标题或内容必须包含关键字
}

var Ppt *Properties

func initProps() {
	Ppt = &Properties{
		Chrome:      `C:\Program Files\Google\Chrome\Application\chrome.exe`,
		PdfChinese:  `wechatarticles/pdf/chinese/webfonts-master`,
		Debug:       true,
		LogFN:       `日志.log`,
		BaseDir:     `D:\`,
		WorkDir:     ``,
		JsonFN:      `内容.json`,
		TJsonFN:     `内容_tmp.json`,
		TxtFN:       ``,
		ExcelFN:     ``,
		PdfFN:       ``,
		SupportMail: true,
		OnlyMail:    false,
		MailUser:    "发送邮件的邮箱用户名",
		MailPwd:     "发送邮件的邮箱用户密码",
		MailSubj:    "",
		MailAuthTO:  []string{"扫码授权的主送邮箱1", "扫码授权的主送邮箱2"},
		MailTO:      []string{"主送邮箱1", "主送邮箱2"},
		MailPdfTO:   []string{"主送邮箱1", "主送邮箱2"},
		MailCC:      []string{"抄送邮箱1", "抄送邮箱2"},
		MailBCC:     []string{"暗送邮箱1", "暗送邮箱2"},
		MailKeys:    []string{"邮件高亮关键字1", "邮件高亮关键字2"},
		WechatUser:  "微信公众号用户名",
		WechatPwd:   "微信公众号用户密码",
		BeginDay:    time.Now().Add(time.Hour * -24).Format("2006-01-02"),
		EdnDay:      time.Now().Add(time.Hour * -24).Format("2006-01-02"),
		Image:       false,
		Sources: []Source{
			{Names: []string{"公众号名称1", "公众号名称2"}, Tag: "公众号名称1和2的分类", MustKeys: []string{"公众号分类的特有邮件高亮关键字1", "公众号分类的特有邮件高亮关键字2"}, MustMatch: false},
			{Names: []string{"公众号名称3", "公众号名称4"}, Tag: "公众号名称3和4的分类", MustKeys: []string{"公众号分类的特有邮件高亮关键字1", "公众号分类的特有邮件高亮关键字2"}, MustMatch: false},
		},
	}

	jsonF, err := os.Open(propFileName)
	if err != nil {
		log.Error("打开配置文件失败，系统将自动生成新的配置文件，请修改参数后重新执行", propFileName, err)
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
		if strings.EqualFold(Ppt.EdnDay, Ppt.BeginDay) {
			Ppt.MailSubj = fmt.Sprintf("公众号信息每日收集：%s(%s)", time.Now().Format("2006-01-02"), Ppt.BeginDay)
		} else {
			Ppt.MailSubj = fmt.Sprintf("公众号信息每日收集：%s(%s至%s)", time.Now().Format("2006-01-02"), Ppt.BeginDay, Ppt.EdnDay)
		}
	}

	if len(Ppt.WorkDir) <= 0 {
		if strings.EqualFold(Ppt.EdnDay, Ppt.BeginDay) {
			Ppt.WorkDir = filepath.Join(Ppt.BaseDir, fmt.Sprintf("爬虫%s(%s)", time.Now().Format("2006-01-02"), Ppt.BeginDay))
		} else {
			Ppt.WorkDir = filepath.Join(Ppt.BaseDir, fmt.Sprintf("爬虫%s(%s_%s)", time.Now().Format("2006-01-02"), Ppt.BeginDay, Ppt.EdnDay))
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
	Cookie      string            `json:"cookie"`
	Token       string            `json:"token"`
	NameFakeIds []NameId          `json:"nameFakeIds"`
	FakeIds     map[string]string `json:"-"`
}

type NameId struct {
	Name   string `json:"name"`
	FakeId string `json:"fakeid"`
	Info   string `json:"signature"`
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
