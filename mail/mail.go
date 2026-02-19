package mail

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"wechatarticles/excel"
	"wechatarticles/http"
	"wechatarticles/log"
	"wechatarticles/pdf"
	"wechatarticles/props"

	"github.com/skip2/go-qrcode"
	"gopkg.in/gomail.v2"
)

// 红色字体替换
const BOLD_PREFIX = `BOLD_PREFIX`
const BOLD_SUFIX = `BOLD_SUFIX`

// 发送日志（后台运行出错，通过邮件发送日志以便进行分析）
func SendLog(subj string) {
	if props.Ppt.SupportMail == false || len(props.Ppt.MailAuthTO) <= 0 {
		return
	}

	message := `
<p>您好</p>

<p style="text-indent:2em">公众号后台作业运行中断，日志见附件</P>

<p style="text-indent:2em">祝好</P>
`
	images := []string{}
	attachments := []string{filepath.Join(props.Ppt.WorkDir, props.Ppt.LogFN), filepath.Join(props.Ppt.WorkDir, props.Ppt.TJsonFN)}

	mailCC := []string{}
	mailBCC := []string{}

	send163(props.Ppt.MailUser, props.Ppt.MailPwd, subj, message, props.Ppt.MailAuthTO, mailCC, mailBCC, images, attachments)
}

// 通过邮件发送登陆二维码实现定时作业远程授权
// 二维码图片保存路径为dir+imageFN
func SendAuth(dir, imageFN string) {

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
	mailCC := []string{}
	mailBCC := []string{}

	send163(props.Ppt.MailUser, props.Ppt.MailPwd, subj, message, props.Ppt.MailAuthTO, mailCC, mailBCC, images, nil)
}

// 爬虫结果需以json文件的形式保存在路径props.Ppt.WorkDir + props.Ppt.JsonFN的文件中
func initResult() ([]*http.Article, []string) {
	//从json文件解析对象
	jsonF, err := os.Open(filepath.Join(props.Ppt.WorkDir, props.Ppt.JsonFN))
	if err != nil {
		log.Error("打开文件失败", err)
	}
	defer jsonF.Close()

	b, err := ioutil.ReadAll(jsonF)
	if err != nil {
		log.Error("读取文件失败：", err)
	}

	var articles []*http.Article
	err = json.Unmarshal(b, &articles)
	if err != nil {
		log.Error("解析json失败", err)
		return nil, nil
	}

	//二维码
	images := make([]string, len(articles), len(articles))
	for n, art := range articles {
		art.QrCodeFN = fmt.Sprintf("%d%s", n, `.png`)
		images[n] = filepath.Join(props.Ppt.WorkDir, art.QrCodeFN)

		qr, err := qrcode.New(art.Link, qrcode.Low)
		if err != nil {
			log.Error("生成二维码失败", err)
		}

		err = qr.WriteFile(128, images[n])
		if err != nil {
			log.Error("生成二维码图片失败", err)
		}
	}
	return articles, images
}

// 通过邮件发送爬虫结果
func SendResult() {
	articles, images := initResult()

	arts := make([]http.Article, len(articles))
	for i, ptr := range articles {
		arts[i] = *ptr
	}

	if len(props.Ppt.ExcelFN) > 0 {
		excel.Write(arts)
	}

	//txt excel
	sendByMail(articles, images)
	//pdf
	sendBySecret(articles)
	//json
	send2Operator()
}

func sendByMail(articles []*http.Article, images []string) {
	//红色字体
	for _, art := range articles {
		set := make(map[string]struct{})
		for _, key := range props.Ppt.MailKeys {
			set[key] = struct{}{}
		}
		for _, src := range props.Ppt.Sources {
			if strings.EqualFold(src.Tag, art.Tag) {
				for _, key := range src.MustKeys {
					set[key] = struct{}{}
				}
			}
		}
		for key := range set {
			art.Title = strings.ReplaceAll(art.Title, key, BOLD_PREFIX+key+BOLD_SUFIX)
			art.Digest = strings.ReplaceAll(art.Digest, key, BOLD_PREFIX+key+BOLD_SUFIX)
			art.Class = strings.ReplaceAll(art.Class, key, BOLD_PREFIX+key+BOLD_SUFIX)
		}
	}

	//邮件正文
	ct, err := template.New("mail").Parse(`
<p>你好</p>

<p style="text-indent:2em">公众号信息汇总如下，扫码查看原文：</P>

<table>
<tr align="left"><th>公众号</th><th>标题</th><th>分类</th><th>摘要</th><th>链接</th></tr>
{{range .}}
<tr align="left"><td width="10%">{{.Source}}</td><td width="30%">{{.Title}}</td><td width="10%">{{.Class}}</td><td width="30%">{{.Digest}}</td><td width="20%"><a href="{{.Link}}">{{.Link}}</a><br><img src="cid:{{.QrCodeFN}}" /></td></tr>
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

	//红色字体
	message := msg.String()
	message = strings.ReplaceAll(message, BOLD_PREFIX, `<font color="#FF0000">`)
	message = strings.ReplaceAll(message, BOLD_SUFIX, `</font>`)

	//附件
	attachments := []string{}
	if len(props.Ppt.TxtFN) > 0 {
		attachments = append(attachments, filepath.Join(props.Ppt.WorkDir, props.Ppt.TxtFN))
	}
	if len(props.Ppt.ExcelFN) > 0 {
		attachments = append(attachments, filepath.Join(props.Ppt.WorkDir, props.Ppt.ExcelFN))
	}

	//邮件发送
	send163(props.Ppt.MailUser, props.Ppt.MailPwd, props.Ppt.MailSubj, message, props.Ppt.MailTO, props.Ppt.MailCC, props.Ppt.MailBCC, images, attachments)
}

// 对于存在邮件拦截的，通过pdf发送
func sendBySecret(articles []*http.Article) {
	if len(props.Ppt.MailPdfTO) == 0 || len(props.Ppt.PdfFN) == 0 {
		return
	}

	//邮件正文
	message := `
<p>你好</p>

<p style="text-indent:2em">公众号信息汇总见附件。</P>


<p style="text-indent:2em">祝好</P>
`

	//红色字体已经由前面的方法处理了
	pdf.Write(articles, BOLD_PREFIX, BOLD_SUFIX)

	//附件
	attachments := []string{filepath.Join(props.Ppt.WorkDir, props.Ppt.PdfFN)}

	//邮件发送
	send163(props.Ppt.MailUser, props.Ppt.MailPwd, props.Ppt.MailSubj, message, props.Ppt.MailPdfTO, props.Ppt.MailCC, props.Ppt.MailBCC, nil, attachments)
}

// 发送管理员
func send2Operator() {
	//邮件正文
	message := `
<p>你好</p>

<p style="text-indent:2em">公众号信息汇总见附件。</P>


<p style="text-indent:2em">祝好</P>
`

	//附件
	attachments := []string{filepath.Join(props.Ppt.WorkDir, props.Ppt.JsonFN)}

	//邮件发送
	send163(props.Ppt.MailUser, props.Ppt.MailPwd, props.Ppt.MailSubj, message, props.Ppt.MailAuthTO, nil, nil, nil, attachments)
}

// go get -v gopkg.in/gomail.v2
func send163(userName, password, subj, message string, mailTo, mailCC, mailBCC, images, attachments []string) {
	// 163 邮箱：
	// SMTP 服务器地址：smtp.163.com（端口：25）
	host := "smtp.163.com"
	port := 25

	m := gomail.NewMessage()

	m.SetAddressHeader("From", userName, "信息搜集") // 增加发件人别名（支持中文）
	//	m.SetHeader("From", userName) // 发件人
	//	m.SetHeader("From", "WechatArticles"+"<"+userName+">") // 增加发件人别名（不支持中文）
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
	log.Info("发送邮件")
	err := d.DialAndSend(m)
	if err != nil {
		log.Error("发送邮件失败", err, subj)
	}
}
