package main

import (
	"bufio"
	"html/template"
	"io"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/russross/blackfriday/v2"
)

// TemplateRenderer is a custom html/template renderer for Echo framework
type templateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *templateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// Add global methods if data is a map
	if viewContext, isMap := data.(map[string]interface{}); isMap {
		viewContext["reverse"] = c.Echo().Reverse
	}
	return t.templates.ExecuteTemplate(w, name, data)
}

// main funcに渡す
func getTemplateRenderer() *templateRenderer {
	fnc := template.FuncMap{
		"toMarkdown": toMarkdown,
		"dtFormat":   dtFormat,
	}
	return &templateRenderer{
		templates: template.Must(template.New("").Funcs(fnc).ParseGlob("templates/*.html")),
	}
}

// markdown convert and no escape (html)
func toMarkdown(str string, isLists bool, uri string, title string) template.HTML {
	// linkにrel="noreferrer noopener", target="_blank"を付与
	//htmlFlag := blackfriday.NoopenerLinks | blackfriday.NoreferrerLinks | blackfriday.HrefTargetBlank
	//renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{Flags: htmlFlag})
	if !isLists {
		//return template.HTML(blackfriday.Run([]byte(str), blackfriday.WithRenderer(renderer)))
		return template.HTML(blackfriday.Run([]byte(str)))
	}
	// <!--more-->が存在する場合は以降の文字列を捨ててリンクを挿入する(WordPress仕様に合わせる)
	buffer := ""
	scanner := bufio.NewScanner(strings.NewReader(str))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, MoreLinkString) {
			buffer += "<a href=\"" + uri + "\">続きを読む<span class=\"srt\">" + title + "</span></a>"
			break
		}
		buffer += line + "\n"
	}
	//return template.HTML(blackfriday.Run([]byte(buffer), blackfriday.WithRenderer(renderer)))
	return template.HTML(blackfriday.Run([]byte(buffer)))
}

// datetime formatter (golangでは何故か具体的な下記日時を指定してyyyy-mm-ddフォーマットをを実現する)(が、mongoでは多分使わない)
func dtFormat(dateTime string) string {
	//return dateTime.Format("2006-01-02")
	return string([]rune(dateTime)[:10])
}
