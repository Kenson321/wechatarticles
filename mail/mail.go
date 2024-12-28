package mail

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/skip2/go-qrcode"
	"gopkg.in/gomail.v2"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"wechatarticles/log"
	"wechatarticles/props"
)

const BOLD_PREFIX = `ABCDEFG`
const BOLD_SUFIX = `HIJKLMN`

type Article struct {
	Source   string `json:"source"`
	Tag      string `json:"tag"`
	Title    string `json:"title"`
	Link     string `json:"link"`
	Time     string `json:"time"`
	Digest   string `json:"digest"`
	Class    string `json:"class"`
	QrCodeFN string `json:"-"`
}

//通过邮件发送登陆二维码实现定时作业远程授权
//二维码图片保存路径为dir+imageFN
//邮箱用户名密码为userName, password
func SendAuth(dir, imageFN, userName, password string) {

	ct, err := template.New("mail").Parse(`
<p>您好</p>

<p style="text-indent:2em">公众号cookie已到期，请重新扫码以便后台作业继续运行</P>

<img src="cid:{{.}}" />

<p style="text-indent:2em">祝好</P>
`)
	if err != nil {
		log.Error("模板分析失败", err)
		return
	}
	msg := new(bytes.Buffer)
	err = ct.Execute(msg, imageFN)
	if err != nil {
		log.Error("转换为html失败", err)
		return
	}
	message := msg.String()

	images := []string{filepath.Join(dir, imageFN)}

	subj := "会话到期"
	mailTo := []string{props.Ppt.MailTO[0]}
	mailCC := []string{props.Ppt.MailBCC[0]}
	mailBCC := []string{}

	send163(userName, password, subj, message, mailTo, mailCC, mailBCC, images, nil)
}

//通过邮件发送爬虫结果
//爬虫结果需以json文件的形式保存在路径dir + jsonFN的文件中
//邮箱用户名密码为userName, password
//附件由attachments指定
func SendResult(dir, jsonFN, userName, password string, attachments []string) {
	jsonF, err := os.Open(filepath.Join(dir, jsonFN))
	if err != nil {
		log.Error("打开文件失败", err)
	}
	defer jsonF.Close()

	b, err := ioutil.ReadAll(jsonF)
	if err != nil {
		log.Error("读取文件失败：", err)
	}

	var articles []*Article
	err = json.Unmarshal(b, &articles)
	if err != nil {
		log.Error("解析json失败", err)
		return
	}

	images := make([]string, len(articles), len(articles))
	for n, art := range articles {
		img := filepath.Join(dir, fmt.Sprintf("%d%s", n, `.png`))
		images[n] = img
		art.QrCodeFN = fmt.Sprintf("%d%s", n, `.png`)

		qr, err := qrcode.New(art.Link, qrcode.Low)
		if err != nil {
			log.Error("生成二维码失败", err)
		}

		err = qr.WriteFile(128, img)
		if err != nil {
			log.Error("生成二维码图片失败", err)
		}

		keys := []string{}
		keys = append(keys, props.Ppt.MailKeys...)
		for _, src := range props.Ppt.Sources {
			if strings.EqualFold(src.Tag, art.Tag) {
				keys = append(keys, src.HighlightMailWords...)
			}
		}
		for _, key := range keys {
			art.Title = strings.ReplaceAll(art.Title, key, BOLD_PREFIX+key+BOLD_SUFIX)
			art.Digest = strings.ReplaceAll(art.Digest, key, BOLD_PREFIX+key+BOLD_SUFIX)
			art.Class = strings.ReplaceAll(art.Class, key, BOLD_PREFIX+key+BOLD_SUFIX)
		}
	}

	ct, err := template.New("mail").Parse(`
<p>你好</p>

<p style="text-indent:2em">公众号信息汇总如下，扫码查看原文：</P>

<table>
<tr align="left"><th>公众号</th><th>标题</th><th>分类</th><th>摘要</th></tr>
{{range .}}
<tr align="left"><td width="10%">{{.Source}}</td><td width="40%">{{.Title}}<br><img src="cid:{{.QrCodeFN}}" /></td><td width="10%">{{.Class}}</td><td width="40%">{{.Digest}}</td></tr>
{{end}}
<table>

<p style="text-indent:2em">祝好</P>
`)
	if err != nil {
		log.Error("模板分析失败", err)
		return
	}
	msg := new(bytes.Buffer)
	err = ct.Execute(msg, articles)
	if err != nil {
		log.Error("转换为html失败", err)
		return
	}

	message := msg.String()
	message = strings.ReplaceAll(message, BOLD_PREFIX, `<font color="#FF0000">`)
	message = strings.ReplaceAll(message, BOLD_SUFIX, `</font>`)

	send163(userName, password, props.Ppt.MailSubj, message, props.Ppt.MailTO, props.Ppt.MailCC, props.Ppt.MailBCC, images, attachments)
}

//go get -v gopkg.in/gomail.v2
func send163(userName, password, subj, message string, mailTo, mailCC, mailBCC, images, attachments []string) {
	// 163 邮箱：
	// SMTP 服务器地址：smtp.163.com（端口：25）
	host := "smtp.163.com"
	port := 25

	m := gomail.NewMessage()

	m.SetHeader("From", userName) // 发件人
	//	m.SetHeader("From", "alias"+"<"+userName+">") // 增加发件人别名
	m.SetHeader("To", mailTo...)   // 收件人，可以多个收件人，但必须使用相同的 SMTP 连接
	m.SetHeader("Cc", mailCC...)   // 抄送，可以多个
	m.SetHeader("Bcc", mailBCC...) // 暗送，可以多个
	m.SetHeader("Subject", subj)   // 邮件主题

	// text/html 的意思是将文件的 content-type 设置为 text/html 的形式，浏览器在获取到这种文件时会自动调用html的解析器对文件进行相应的处理。
	// 可以通过 text/html 处理文本格式进行特殊处理，如换行、缩进、加粗等等
	m.SetBody("text/html", message)
	// text/plain的意思是将文件设置为纯文本的形式，浏览器在获取到这种文件时并不会对其进行处理
	// m.SetBody("text/plain", "纯文本")

	for _, img := range images {
		m.Embed(img) //图片
	}

	for _, att := range attachments {
		m.Attach(att) // 附件文件，可以是文件，照片，视频等等
	}

	d := gomail.NewDialer(
		host,
		port,
		userName,
		password,
	)
	// 关闭SSL协议认证
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	err := d.DialAndSend(m)
	if err != nil {
		log.Error("发送邮件失败", err, subj)
	}
}