package layout

import (
	"encoding/json"
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"path/filepath"
	"strings"
)

type OfficialService struct{}

type OfficialCheckValue struct {
	Proxy  string
	Domain string
}

var OfficialCheckConfig = new(OfficialCheckValue)

func (service *OfficialService) OfficialInit() Composite {
	return Composite{
		Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
		Layout:     HBox{},
		Children: []Widget{
			PushButton{Text: "官方认证", OnClicked: func() {
				service.OfficialCheckConfigLoad() // 加载本地配置
				if cmd, err := service.OfficialCheckDialog(windowMain, OfficialCheckConfig); err != nil {
					new(WindowCustom).ConsoleErron(fmt.Sprintf(`弹窗失败：失败原因：%s`, err.Error()))
				} else if cmd == walk.DlgCmdOK {
					service.OfficialCheckConfigSet()
				}
			}},
			PushButton{Text: "开始", OnClicked: func() {
				service.OfficialCheckConfigLoad() // 加载本地配置
				if service.OfficialCheckConfigSet() {
					go service.OfficialCheckStart()
				}
			}, MaxSize: Size{Width: 50}},
		},
	}
}

func (service *OfficialService) OfficialCheckStart() {
	domain := strings.Split(OfficialCheckConfig.Domain, "\r\n")
	for _, obj := range domain {
		service.OfficialRequest(obj, false)
	}
	new(WindowCustom).ConsoleSuccess(fmt.Sprintf(`%d个域名查询全部完成！请注意查看控制台数据~`, len(domain)))
}

func (service *OfficialService) OfficialRequest(domain string, proxy bool) {
	urls := "https://www.baidu.com/s"
	body := new(RequestService).RespOfficial(urls, domain).requestCustom(proxy, OfficialCheckConfig.Proxy)
	if len(body) == 0 || string(body) == "" {
		new(WindowCustom).ConsoleErron(fmt.Sprintf(`域名[%s]查询失败；重试中。。。`, domain))
		service.OfficialRequest(domain, true)
	}
	fmt.Println(string(body))
}

func (service *OfficialService) OfficialCheckConfigSet() bool {
	var context string
	var maps map[string]string
	bs, _ := filepath.Abs("config")
	mjson, _ := json.Marshal(OfficialCheckConfig)
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
	config := fmt.Sprintf(`%s\official.config`, bs)
	return new(RequestService).CreateConfig(config, context)
}

func (service *OfficialService) OfficialCheckDialog(owner *walk.MainWindow, officialcheck *OfficialCheckValue) (int, error) {
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
		DataBinder:    DataBinder{AssignTo: &db, Name: "officialcheck", DataSource: officialcheck, ErrorPresenter: ToolTipErrorPresenter{}},
		Children: []Widget{
			Composite{Layout: Grid{Columns: 2}, Children: service.DialogFieldForm()},
			Composite{Layout: HBox{}, Children: []Widget{Composite{Layout: Grid{Columns: 2}, Children: []Widget{PushButton{AssignTo: &acceptPB, Text: "确定", OnClicked: func() { _ = db.Submit(); dlg.Accept() }}, PushButton{AssignTo: &cancelPB, Text: "取消", OnClicked: func() { dlg.Cancel() }}}}}},
		},
	}.Run(owner)
}

func (service *OfficialService) DialogFieldForm() []Widget {
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

func (service *OfficialService) DialogField() map[string]string {
	return map[string]string{
		"Proxy":  "小象代理地址",
		"Domain": "等待查询域名",
	}
}

func (service *OfficialService) OfficialCheckConfigLoad() {
	bs, _ := filepath.Abs("config")
	config := fmt.Sprintf(`%s\official.config`, bs)
	cgf := new(RequestService).TextConfig(config)
	if _, ok := cgf["Domain"]; ok {
		cgf["Domain"] = strings.Replace(cgf["Domain"], "$", "\r\n", -1)
	}
	mjson, _ := json.Marshal(cgf)
	mjson = []byte(strings.Replace(string(mjson), "#", ":", -1))
	_ = json.Unmarshal(mjson, &OfficialCheckConfig)
}
