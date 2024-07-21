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
	"strings"
	"sync"
	"time"
	"wechatarticles/log"
)

func Visit(bin, baseDir, url string, onlyText bool) (content string) {
	l := launcher.New().Headless(true).Bin(bin)
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

	dir := baseDir + title
	os.MkdirAll(dir, os.ModeDir|os.ModePerm)
	file, err := os.Create(baseDir + title + ".md")
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

//深度遍历
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
			imgF := fmt.Sprintf("%s\\%d.png", dir, i)
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

func GetAuth(bin, usr, pwd string) (token, cookie string) {
	l := launcher.New().Headless(false).Bin(bin)
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
			log.Info("cookie: %s", cookie)
			
			token = ctx.Request.URL().String()
			i := strings.LastIndex(token, "token=")
			token = token[i+6:]
			i = strings.IndexRune(token, '&')
			token = token[:i]
			log.Info("token: %s", token)
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
	el.MustInput(usr)
	el = page.MustElement("#header > div.banner > div > div > div.login__type__container.login__type__container__account > form > div.login_input_panel > div:nth-child(2) > div > span > input")
	el.MustInput(pwd)
	el = page.MustElement("#header > div.banner > div > div > div.login__type__container.login__type__container__account > form > div.login_btn_panel > a")
	el.MustClick()
	w.Wait()
	page.MustWaitStable()

	return
}
