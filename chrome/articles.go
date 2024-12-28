package chrome

import (
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"wechatarticles/log"
	"wechatarticles/props"
	"wechatarticles/mail"
)

const authpng = `auth.png`

//获取url内的文章内容，onlyText为true表示只获取文字不获取图片，数据保存在props.Ppt.WorkDir目录下
func Visit(url string, onlyText bool) (content string) {
	l := launcher.New().Headless(true) //不打开浏览器
	_, err := os.Stat(props.Ppt.Chrome)
	if err != nil {
		log.Error("未指定或未找到chrome执行程序：", props.Ppt.Chrome)
	} else {
		l.Bin(props.Ppt.Chrome) 
	}
	cc := l.MustLaunch()
	browser := rod.New().ControlURL(cc).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage(url)
	page.MustWaitStable()

	exists, el, err := page.HasX(`//*[@id="activity-name"]`)
	if err != nil {
		log.Error("获取标题元素失败", url, err)
		return
	}
	if !exists {
		log.Error("获取标题元素失败", url)
		return
	}
	title := el.MustText()

	title = strings.ReplaceAll(title, " ", "_")
	title = strings.ReplaceAll(title, "|", "_")
	title = strings.ReplaceAll(title, `:`, "_")
	title = strings.ReplaceAll(title, `"`, "_")
	title = strings.ReplaceAll(title, `?`, "_")
	title = strings.ReplaceAll(title, `*`, "_")
	title = strings.ReplaceAll(title, `/`, "_")
	title = strings.ReplaceAll(title, `\`, "_")
	title = strings.ReplaceAll(title, `<`, "_")
	title = strings.ReplaceAll(title, `>`, "_")
	log.Info(title)
	log.Info(url)

	dir := filepath.Join(props.Ppt.WorkDir, title)
	os.MkdirAll(dir, os.ModeDir|os.ModePerm)
	file, err := os.Create(filepath.Join(props.Ppt.WorkDir, title + ".md"))
	if err != nil {
		if err != nil {
			log.Error("生成文件失败", err)
		}
	}
	defer file.Close()
	fmt.Fprintln(file, title)
	fmt.Fprintln(file, url)
	fmt.Fprintln(file, "")

	//一次获取所有文字
	el = page.MustElement("#js_article")
	content = el.MustText()
	content = strings.Join(strings.Fields(content), " ")

	if onlyText {
		fmt.Fprintln(file, content)
	} else {
		els := page.MustElementsX("//div[@id='js_content']/*")
		deepVisit(els[0], file, dir, title)
	}

	return
}

var repeat string

//深度遍历，以支持获取图片和保持顺序
//文字保存在f所代表的markdown文件中
//图片保存在dir目录下
//title为目录相对路径名，用于在markdown文档中引用图片
func deepVisit(e *rod.Element, f *os.File, dir string, title string) {
	log.Debug(e.String())

	text := e.MustText()
	text = strings.TrimSpace(text)
	if len(text) > 0 && !strings.Contains(repeat, text) {
		log.Info("文字：", text)
		fmt.Fprintln(f, text)
		repeat = text
	}

	if strings.Contains(e.String(), "<img") {
		var b []byte
		s, _ := e.Attribute("src")
		log.Info("图片：", *s)
		if strings.HasPrefix(*s, "http") {
			b = e.MustResource()
		} else {
			s, _ = e.Attribute("data-src")
			if s == nil {
				s = new(string)
			}
			log.Info("图片：", *s)
			if strings.HasPrefix(*s, "http") {
				res, err := http.Get(*s)
				if err != nil {
					log.Error("http发送失败：", err)
				}
				defer res.Body.Close()
				b, err = ioutil.ReadAll(res.Body)
				if err != nil {
					log.Error("http读取结果失败：", err)
				}
			}
		}
		if len(b) > 0 {
			rand.Seed(time.Now().UnixNano())
			i := rand.Int31()
			imgF := filepath.Join(dir, fmt.Sprintf("%d.png", i))
			err := utils.OutputFile(imgF, b)
			if err != nil {
				log.Error("生成图片失败：", err)
			} else {
				log.Info("生成图片：", imgF)
			}
			fmt.Fprintf(f, "![%d](.\\%s\\%d.jpg)\n", i, title, i)
			fmt.Fprintf(f, "%s\n", *s)
		}
	}

	ne, err := e.ElementX("*")
	if err != nil {
		//return
	} else {
		deepVisit(ne, f, dir, title)
	}

	ne, err = e.Next()
	if err != nil {
		//return
	} else {
		deepVisit(ne, f, dir, title)
	}
}

//模拟登陆微信公众号平台
//由于需要扫码登陆，可以在windows平台下打开浏览器，或者保存登陆二维码图片到当前目录下，通过打开图片扫码，又或者通过发送邮件的方式通知用户扫码授权
func GetAuth() (token, cookie string) {
	l := launcher.New()
	_, err := os.Stat(props.Ppt.Chrome)
	if err != nil {
		log.Error("未指定或未找到chrome执行程序：", props.Ppt.Chrome)
		l.Headless(true)
	} else {
		l.Headless(false).Bin(props.Ppt.Chrome) //打开浏览器以便扫码登陆
	}
	cc := l.MustLaunch()
	browser := rod.New().ControlURL(cc).MustConnect()
	defer browser.MustClose()

	var w sync.WaitGroup
	w.Add(1)
	router := browser.HijackRequests()
	f := func(ctx *rod.Hijack) {
		ctx.MustLoadResponse()

		req := ctx.Request.Req()
		cookie = fmt.Sprintf("%s", req.Header["Cookie"])
		cookie = cookie[1:]
		l := len(cookie)
		cookie = cookie[:l-1]

		token = ctx.Request.URL().String()
		i := strings.LastIndex(token, "token=")
		token = token[i+6:]
		i = strings.IndexRune(token, '&')
		token = token[:i]
		
		w.Done()
	}
	router.MustAdd("*/appmsgpublish*", f)
	go router.Run()

	page := browser.MustPage("https://mp.weixin.qq.com/")
	page.MustWindowFullscreen()
	page.MustWaitStable()

	el := page.MustElement("#header > div.banner > div > div > div.login__type__container.login__type__container__scan > a")
	el.MustClick()
	el = page.MustElement("#header > div.banner > div > div > div.login__type__container.login__type__container__account > form > div.login_input_panel > div:nth-child(1) > div > span > input")
	el.MustInput(props.Ppt.WechatUser)
	el = page.MustElement("#header > div.banner > div > div > div.login__type__container.login__type__container__account > form > div.login_input_panel > div:nth-child(2) > div > span > input")
	el.MustInput(props.Ppt.WechatPwd)
	el = page.MustElement("#header > div.banner > div > div > div.login__type__container.login__type__container__account > form > div.login_btn_panel > a")
	el.MustClick()

	page.MustWaitStable()
	time.Sleep(time.Second * 3)
	
	log.Info("如果没有打开浏览器，可以打开本地文件扫码，或接收邮件扫码", authpng)
	page.MustScreenshot(authpng)
	if props.Ppt.SupportMail == true {
		mail.SendAuth(`.`, authpng, props.Ppt.MailUser, props.Ppt.MailPwd)
	}

	w.Wait()
	//	page.MustWaitStable()

	return
}
