package layout

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"path/filepath"
	"strings"
)

type IcpCheckService struct{}

type IcpCheckValue struct {
	Proxy  string
	Domain string
}

var IcpCheckConfig = new(IcpCheckValue)

func (service *IcpCheckService) IcpCheckInit() Composite {
	return Composite{
		Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
		Layout:     HBox{},
		Children: []Widget{
			PushButton{Text: "备案查询", OnClicked: func() {
				service.IcpCheckConfigLoad() // 加载本地配置
				if cmd, err := service.IcpCheckDialog(windowMain, IcpCheckConfig); err != nil {
					new(WindowCustom).ConsoleErron(fmt.Sprintf(`弹窗失败：失败原因：%s`, err.Error()))
				} else if cmd == walk.DlgCmdOK {
					service.IcpCheckConfigSet()
				}
			}},
			PushButton{Text: "开始", OnClicked: func() {
				service.IcpCheckConfigLoad() // 加载本地配置
				if service.IcpCheckConfigSet() {
					go service.IcpCheckStart()
				}
			}, MaxSize: Size{Width: 50}},
		},
	}
}

func (service *IcpCheckService) IcpCheckStart() {
	domain := strings.Split(IcpCheckConfig.Domain, "\r\n")
	for _, obj := range domain {
		service.ReqStart(obj, false)
	}
}

func (service *IcpCheckService) ReqStart(domain string, proxy bool) {
	urls := fmt.Sprintf(`https://www.beianx.cn/search/%s`, domain)
	body := new(RequestService).RespHtml(urls).requestCustom(proxy, IcpCheckConfig.Proxy)
	if len(body) == 0 || string(body) == "" {
		new(WindowCustom).ConsoleErron(fmt.Sprintf(`域名[%s]查询失败；重试中。。。`, domain))
		service.ReqStart(domain, true)
	}
	var respStr string
	if dom, err := goquery.NewDocumentFromReader(strings.NewReader(string(body))); err == nil {
		dom.Find(".table-bordered").Find("tbody").Find("tr").Find("td").Each(func(i int, selection *goquery.Selection) {
			html, _ := selection.Html()
			if strings.Contains(new(RequestService).TrimHtml(html), "没有查询到记录") == false && i <= 7 {
				if i == 3 {
					respStr = fmt.Sprintf(`备案域名:[%s];备案号：[%s]`, domain, new(RequestService).TrimHtml(html))
				}
			}
		})
	}
	if respStr != "" {
		new(WindowCustom).ConsoleSuccess(respStr)
	} else {
		new(WindowCustom).ConsoleErron(fmt.Sprintf(`域名[ %s ]未备案~`, domain))
	}
}

func (service *IcpCheckService) IcpCheckDialog(owner *walk.MainWindow, icpcheck *IcpCheckValue) (int, error) {
	var dlg *walk.Dialog
	var db *walk.DataBinder
	var acceptPB, cancelPB *walk.PushButton
	return Dialog{
		AssignTo:      &dlg,
		Title:         "域名备案批量查询",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		MinSize:       Size{Width: 400},
		Layout:        VBox{},
		DataBinder:    DataBinder{AssignTo: &db, Name: "icpcheck", DataSource: icpcheck, ErrorPresenter: ToolTipErrorPresenter{}},
		Children: []Widget{
			Composite{Layout: Grid{Columns: 2}, Children: service.DialogFieldForm()},
			Composite{Layout: HBox{}, Children: []Widget{Composite{Layout: Grid{Columns: 2}, Children: []Widget{PushButton{AssignTo: &acceptPB, Text: "确定", OnClicked: func() { _ = db.Submit(); dlg.Accept() }}, PushButton{AssignTo: &cancelPB, Text: "取消", OnClicked: func() { dlg.Cancel() }}}}}},
		},
	}.Run(owner)
}

func (service *IcpCheckService) DialogField() map[string]string {
	return map[string]string{
		"Proxy":  "小象代理地址",
		"Domain": "等待查询域名",
	}
}

func (service *IcpCheckService) DialogFieldForm() []Widget {
	var display []Widget
	for index, val := range service.DialogField() {
		if index == "Domain" {
			display = append(display, Label{Text: fmt.Sprintf(`%s：`, val)}, TextEdit{
				Text:    Bind(index),
				MinSize: Size{Height: 200},
				VScroll: true,
			})
		} else {
			display = append(display, Label{Text: fmt.Sprintf(`%s：`, val)}, LineEdit{Text: Bind(index)})
		}
	}
	return display
}

func (service *IcpCheckService) IcpCheckConfigLoad() {
	bs, _ := filepath.Abs("config")
	config := fmt.Sprintf(`%s\icpcheck.config`, bs)
	cgf := new(RequestService).TextConfig(config)
	if _, ok := cgf["Domain"]; ok {
		cgf["Domain"] = strings.Replace(cgf["Domain"], "$", "\r\n", -1)
	}
	mjson, _ := json.Marshal(cgf)
	mjson = []byte(strings.Replace(string(mjson), "#", ":", -1))
	_ = json.Unmarshal(mjson, &IcpCheckConfig)
}

func (service *IcpCheckService) IcpCheckConfigSet() bool {
	var context string
	var maps map[string]string
	bs, _ := filepath.Abs("config")
	mjson, _ := json.Marshal(IcpCheckConfig)
	_ = json.Unmarshal(mjson, &maps)
	for index, val := range service.DialogField() {
		if _, ok := maps[index]; ok == false || len([]rune(maps[index])) <= 0 {
			new(WindowCustom).ConsoleErron(fmt.Sprintf(`配置错误；请正确配置[%s]的值;`, val))
			return false
		} else {
			text := strings.Replace(maps[index], ":", "#", -1)
			text = strings.Replace(text, "\r\n", "$", -1)
			context += fmt.Sprintf("%s:%s\r\n", index, text)
		}
	}
	config := fmt.Sprintf(`%s\icpcheck.config`, bs)
	return new(RequestService).CreateConfig(config, context)
}
