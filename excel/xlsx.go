package excel

import (
	"github.com/tealeg/xlsx"
	"wechatarticles/http"
	"wechatarticles/log"
	"wechatarticles/props"
	"path/filepath"
)

func Write(articles []http.Article) {
	file := xlsx.NewFile()

	sheet, err := file.AddSheet("result")
	if err != nil {
		log.Error("excel失败", err)
		return
	}

	row := sheet.AddRow()

	cell := row.AddCell()
	cell.Value = "Source"
	cell = row.AddCell()
	cell.Value = "Title"
	cell = row.AddCell()
	cell.Value = "Link"
	cell = row.AddCell()
	cell.Value = "Content"

	for _, art := range articles {
		row := sheet.AddRow()

		cell := row.AddCell()
		cell.Value = art.Source
		cell = row.AddCell()
		cell.Value = art.Title
		cell = row.AddCell()
		//cell.Value = art.Link
		cell.SetFormula(`=HYPERLINK("`+art.Link+`", "`+art.Link+`")`)
		cell = row.AddCell()
		cell.Value = art.Content
	}

	err = file.Save(filepath.Join(props.Ppt.WorkDir, props.Ppt.ExcelFN))
	if err != nil {
		log.Error("excel失败", err)
		return
	}
}
