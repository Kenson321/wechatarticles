package pdf

import (
	"path/filepath"
	"regexp"
	"wechatarticles/http"
	"wechatarticles/log"
	"wechatarticles/props"

	"strings"

	"github.com/jung-kurt/gofpdf"
)

var lh float64 = 8  //行高，大于0才会换行
var nh float64 = 15 //换行高

func Write(articles []*http.Article, boldPre, boldSuf string) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	//wget https://github.com/jsntn/webfonts/archive/refs/heads/master.zip
	pdf.SetFontLocation(props.Ppt.PdfChinese)
	pdf.AddUTF8Font("中文", "", "NotoSansSC-Regular.ttf")
	pdf.SetFont("中文", "", 16)

	for _, art := range articles {
		line(pdf, art.Source, boldPre, boldSuf)
		pdf.Ln(8)
		line(pdf, art.Title, boldPre, boldSuf)
		pdf.Bookmark("-"+art.Title, 0, -1)
		pdf.Ln(8)
		if len(art.Class) > 0 {
			line(pdf, art.Class, boldPre, boldSuf)
			pdf.Ln(8)
		}
		if len(art.Digest) > 0 {
			line(pdf, art.Digest, boldPre, boldSuf)
			pdf.Ln(8)
		}
		pdf.SetTextColor(0, 0, 200)
		pdf.WriteLinkString(lh, art.Link, art.Link)
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(8)
		pdf.ImageOptions(filepath.Join(props.Ppt.WorkDir, art.QrCodeFN), 50, pdf.GetY(), 40, 0, true, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
		pdf.Ln(nh)
	}

	err := pdf.OutputFileAndClose(filepath.Join(props.Ppt.WorkDir, props.Ppt.PdfFN))
	if err != nil {
		log.Error("生成pdf失败", err)
		return
	}
}

func line(pdf *gofpdf.Fpdf, content, boldPre, boldSuf string) {
	txt := replaceSpecialChars(content)
	blacks := strings.Split(txt, boldPre)
	pdf.Write(lh, "-"+blacks[0])
	for i := 1; i < len(blacks); i++ {
		reds := strings.Split(blacks[i], boldSuf)
		if len(reds) != 2 {
			log.Error("红色字体替换逻辑有问题，原文:" + txt + ", 问题字段:" + blacks[i])
		}
		pdf.SetTextColor(200, 0, 0)
		pdf.Write(lh, reds[0])
		pdf.SetTextColor(0, 0, 0)
		pdf.Write(lh, reds[1])
	}
}

// 替换pdf打印不出的特殊字符为？
func replaceSpecialChars(s string) string {
	//非:空白、中文、英文、数字、标点符号、符号——这个有问题
	//	re := regexp.MustCompile(`[^\s\p{Han}a-zA-Z0-9\p{P}\p{S}]`)
	//非:空白、中文、英文、数字、标点符号、数学符号
	//	re := regexp.MustCompile(`[^\s\p{Han}a-zA-Z0-9\p{P}\p{Sm}]`)
	//其它符号
	re := regexp.MustCompile(`[\p{So}]`)
	ns := re.ReplaceAllString(s, "？")
	return ns
}
