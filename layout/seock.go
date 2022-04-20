package layout

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"path/filepath"
	"strconv"
	"strings"
)

type SeoCheckService struct{}

type SeoCheckValue struct {
	Proxy  string
	Domain string
}

var SeoCheckConfig = new(SeoCheckValue)

func (service *SeoCheckService) SeoCheckInit() Composite {
	return Composite{
		Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
		Layout:     HBox{},
		Children: []Widget{
			PushButton{Text: "权重查询", OnClicked: func() {
				service.SeoCheckConfigLoad() // 加载本地配置
				if cmd, err := service.SeoCheckDialog(windowMain, SeoCheckConfig); err != nil {
					new(WindowCustom).ConsoleErron(fmt.Sprintf(`弹窗失败：失败原因：%s`, err.Error()))
				} else if cmd == walk.DlgCmdOK {
					service.SeoCheckConfigSet()
				}
			}},
			PushButton{Text: "开始", OnClicked: func() {
				service.SeoCheckConfigLoad() // 加载本地配置
				if service.SeoCheckConfigSet() {
					go service.SeoCheckStart()
				}
			}, MaxSize: Size{Width: 50}},
		},
	}
}

func (service *SeoCheckService) SeoCheckStart() {
	domain := strings.Split(SeoCheckConfig.Domain, "\r\n")
	for _, obj := range domain {
		var resp = map[string]int{}
		for _, site := range []string{
			fmt.Sprintf(`https://rank.chinaz.com/%s`, obj),
			fmt.Sprintf(`https://rank.chinaz.com/sorank/%s`, obj),
			fmt.Sprintf(`https://rank.chinaz.com/sogoupc/%s`, obj),
			fmt.Sprintf(`https://rank.chinaz.com/smrank/%s`, obj),
			fmt.Sprintf(`https://rank.chinaz.com/toutiao/%s`, obj),
		} {
			for key, ob := range service.ReqStart(obj, false, site) {
				resp[key] = ob
			}
		}
		service.sendData(resp, obj)
	}
}

func (service *SeoCheckService) sendData(d map[string]int, o string) {
	var isok bool
	var message string
	for k, n := range d {
		if n > 0 {
			isok = true
		}
		message += fmt.Sprintf(`[ %s:%d ]`, k, n)
	}
	message = fmt.Sprintf(`域名：%s 结果 %s`, o, message)
	if isok {
		new(WindowCustom).ConsoleSuccess(message)
	} else {
		new(WindowCustom).ConsoleErron(message)
	}
}

func (service *SeoCheckService) tagIcon(tag string) (string, int) {
	var maps = map[string]string{
		"baidu":   "百度PC",
		"bd":      "百度移动",
		"360":     "360",
		"sogou":   "搜狗",
		"shenma":  "神马",
		"toutiao": "头条",
	}
	for key, obj := range maps {
		if strings.Contains(tag, key) {
			if num, err := strconv.Atoi(strings.Replace(tag, key, "", -1)); err == nil && num > 0 {
				return obj, num
			} else {
				return obj, 0
			}
		}
	}
	return "", 0
}

func (service *SeoCheckService) ReqStart(domain string, proxy bool, urls string) map[string]int {
	body := new(RequestService).RespHtmlSeo(urls, domain).requestCustom(proxy, SeoCheckConfig.Proxy)
	if len(body) == 0 || string(body) == "" {
		new(WindowCustom).ConsoleErron(fmt.Sprintf(`域名[%s]查询失败；重试中。。。`, domain))
		service.ReqStart(domain, true, urls)
	}
	var resp = map[string]int{}
	if dom, err := goquery.NewDocumentFromReader(strings.NewReader(string(body))); err == nil {
		dom.Find("._chinaz-rank-nc").Find("ul").Each(func(i int, selection *goquery.Selection) {
			selection.Each(func(i int, se *goquery.Selection) {
				px := `//csstools.chinaz.com/tools/images/rankicons/`
				if src, _ := se.Find("img").Attr("src"); src != "" && strings.Contains(src, px) {
					src = strings.Replace(strings.Replace(src, px, "", -1), ".png", "", -1)
					if title, num := service.tagIcon(src); title != "" {
						resp[title] = num
					}
				}
			})
		})
	}
	return resp
}

func (service *SeoCheckService) SeoCheckConfigLoad() {
	bs, _ := filepath.Abs("config")
	config := fmt.Sprintf(`%s\seocheck.config`, bs)
	cgf := new(RequestService).TextConfig(config)
	if _, ok := cgf["Domain"]; ok {
		cgf["Domain"] = strings.Replace(cgf["Domain"], "$", "\r\n", -1)
	}
	mjson, _ := json.Marshal(cgf)
	mjson = []byte(strings.Replace(string(mjson), "#", ":", -1))
	_ = json.Unmarshal(mjson, &SeoCheckConfig)
}

func (service *SeoCheckService) SeoCheckDialog(owner *walk.MainWindow, seocheck *SeoCheckValue) (int, error) {
	var dlg *walk.Dialog
	var db *walk.DataBinder
	var acceptPB, cancelPB *walk.PushButton
	return Dialog{
		AssignTo:      &dlg,
		Title:         "域名权重批量查询",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		MinSize:       Size{Width: 400},
		Layout:        VBox{},
		DataBinder:    DataBinder{AssignTo: &db, Name: "seocheck", DataSource: seocheck, ErrorPresenter: ToolTipErrorPresenter{}},
		Children: []Widget{
			Composite{Layout: Grid{Columns: 2}, Children: service.DialogFieldForm()},
			Composite{Layout: HBox{}, Children: []Widget{Composite{Layout: Grid{Columns: 2}, Children: []Widget{PushButton{AssignTo: &acceptPB, Text: "确定", OnClicked: func() { _ = db.Submit(); dlg.Accept() }}, PushButton{AssignTo: &cancelPB, Text: "取消", OnClicked: func() { dlg.Cancel() }}}}}},
		},
	}.Run(owner)
}

func (service *SeoCheckService) DialogFieldForm() []Widget {
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

func (service *SeoCheckService) DialogField() map[string]string {
	return map[string]string{
		"Proxy":  "小象代理地址",
		"Domain": "等待查询域名",
	}
}

func (service *SeoCheckService) SeoCheckConfigSet() bool {
	var context string
	var maps map[string]string
	bs, _ := filepath.Abs("config")
	mjson, _ := json.Marshal(SeoCheckConfig)
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
	config := fmt.Sprintf(`%s\seocheck.config`, bs)
	return new(RequestService).CreateConfig(config, context)
}
