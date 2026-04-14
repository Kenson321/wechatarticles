package chrome

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"wechatarticles/log"
	"wechatarticles/mail"
	"wechatarticles/props"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
)

const authpng = `auth.png`

// 获取url内的文章内容，数据保存在props.Ppt.WorkDir目录下的新目录中
func Visit(url string) (content string) {
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

	page.WaitStable(time.Second * 5)

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
	log.Info("格式化后的标题：", title)

	dir := filepath.Join(props.Ppt.WorkDir, title)
	os.MkdirAll(dir, os.ModeDir|os.ModePerm)
	file, err := os.Create(filepath.Join(props.Ppt.WorkDir, title+".md"))
	if err != nil {
		if err != nil {
			log.Error("生成文件失败", err)
		}
	}
	defer file.Close()
	fmt.Fprintln(file, title)
	fmt.Fprintln(file, url)
	fmt.Fprintln(file, "")

	if props.Ppt.Image {
		els := page.MustElementsX("//div[@id='js_content']/*")
		deepVisit(els[0], file, dir, title)
	} else {
		//一次获取所有文字
		el = page.MustElement("#js_article")
		content = el.MustText()
		content = strings.Join(strings.Fields(content), " ")
		fmt.Fprintln(file, content)
	}

	return
}

var repeat string

// 深度遍历，以支持获取图片和保持顺序
// 文字保存在f所代表的markdown文件中
// 图片保存在dir目录下
// title为目录相对路径名，用于在markdown文档中引用图片
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
		image(e, f, dir)
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

// 下载图片
func image(e *rod.Element, f *os.File, dir string) {
	var b0, b1, b2 []byte

	src, _ := e.Attribute("src")
	if src == nil {
		src = new(string)
	}
	log.Info("图片：", *src)

	if strings.HasPrefix(*src, "http") {
		//tp=webp 替换为 tp=nowebp
		//		if strings.Contains(newSrc, "tp=webp") {
		//			newSrc = strings.ReplaceAll(newSrc, "tp=webp", "tp=nowebp")
		//			log.Info("图片新地址：", newSrc)
		//		} else {
		b1 = e.MustResource()
		//		}
	}

	dataSrc, _ := e.Attribute("data-src")
	if dataSrc == nil {
		dataSrc = new(string)
	}
	log.Info("图片：", *dataSrc)
	if strings.Contains(*dataSrc, "tp=webp") {
		tsrc := strings.ReplaceAll(*dataSrc, "tp=webp", "tp=nowebp")
		dataSrc = &tsrc
		log.Info("图片新地址：", *dataSrc)
	}

	if strings.HasPrefix(*dataSrc, "https") {
		req, err := http.NewRequest("GET", *dataSrc, nil)
		if err != nil {
			log.Error("http发送失败：", err)
		} else {
			tls11Transport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
			client := &http.Client{
				Transport: tls11Transport,
			}
			res, err := client.Do(req)
			if err != nil {
				log.Error("http发送失败：", err)
			} else {
				defer res.Body.Close()
				b2, err = ioutil.ReadAll(res.Body)
				if err != nil {
					log.Error("http读取结果失败：", err)
				}
			}
		}
	} else if strings.HasPrefix(*dataSrc, "http") {
		res, err := http.Get(*dataSrc)
		if err != nil {
			log.Error("http发送失败：", err)
		} else {
			defer res.Body.Close()
			b2, err = ioutil.ReadAll(res.Body)
			if err != nil {
				log.Error("http读取结果失败：", err)
			}
		}
	}

	if len(b1) > len(b2) {
		b0 = b1
	} else {
		b0 = b2
	}
	if len(b0) > 0 {
		rand.Seed(time.Now().UnixNano())
		i := rand.Int31()
		imgF := filepath.Join(dir, fmt.Sprintf("%d.jpg", i))
		err := utils.OutputFile(imgF, b0)
		if err != nil {
			log.Error("生成图片失败：", err)
		} else {
			log.Info("生成图片：", imgF)
		}
		fmt.Fprintf(f, "![%d](.\\resource\\%d.jpg)\n", i, i)
		fmt.Fprintf(f, "%s\n", *src)
		fmt.Fprintf(f, "%s\n", *dataSrc)
	}
}

// 模拟登陆微信公众号平台
// 由于需要扫码登陆，可以在windows平台下打开浏览器，或者保存登陆二维码图片到当前目录下，通过打开图片扫码，又或者通过发送邮件的方式通知用户扫码授权
func GetAuth() (token, cookie string) {
	l := launcher.New()
	_, err := os.Stat(props.Ppt.Chrome)
	if err != nil {
		log.Error("未指定或未找到chrome执行程序：", props.Ppt.Chrome)
		l.Headless(true)
	} else {
		if len(props.Ppt.MailAuthTO) > 0 { //发送邮件则不打开浏览器扫码登陆
			l.Headless(true).Bin(props.Ppt.Chrome)
		} else { //打开浏览器以便扫码登陆
			l.Headless(false).Bin(props.Ppt.Chrome)
		}
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
	if props.Ppt.SupportMail == true && len(props.Ppt.MailAuthTO) > 0 {
		mail.SendAuth(`.`, authpng)
	}

	w.Wait()
	//	page.MustWaitStable()

	return
}
