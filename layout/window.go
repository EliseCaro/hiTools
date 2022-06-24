package layout

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	"strings"
	"time"
)

var windowMain *walk.MainWindow

var outTESuccess *walk.TextEdit

var outTEErron *walk.TextEdit

var ProxyCore string

type WindowCustom struct{}

// ConsoleSuccess 控制台信息输出
func (layout *WindowCustom) ConsoleSuccess(str string) {
	var text = strings.Split(outTESuccess.Text(), "\r\n")
	var date = fmt.Sprintf(`%s => `, time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05"))
	var strs = fmt.Sprintf("%s%s\r\n", date, str)
	for index, val := range text {
		if index < 1000 && val != "" {
			strs += fmt.Sprintf("%s\r\n", val)
		}
	}
	_ = outTESuccess.SetText(strs)
}

// ConsoleErron 控制台信息输出
func (layout *WindowCustom) ConsoleErron(str string) {
	var text = strings.Split(outTEErron.Text(), "\r\n")
	var date = fmt.Sprintf(`%s => `, time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05"))
	var strs = fmt.Sprintf("%s%s\r\n", date, str)
	for index, val := range text {
		if index < 20 && val != "" {
			strs += fmt.Sprintf("%s\r\n", val)
		}
	}
	_ = outTEErron.SetText(strs)
}

// ToolsLeftDisplay 左侧布局
func (layout *WindowCustom) ToolsLeftDisplay() Composite {
	return Composite{
		Layout: VBox{},
		Children: []Widget{
			new(YpYunService).CropImage(),
			new(IcpCheckService).IcpCheckInit(),
			new(SeoCheckService).SeoCheckInit(),
			new(OfficialService).OfficialInit(),
			new(BaiduService).BaiduSdk(),
		},
	}
}

// ToolsRightDisplay 右侧布局
func (layout *WindowCustom) ToolsRightDisplay() Composite {
	return Composite{
		Layout: VBox{},
		Children: []Widget{
			TextEdit{
				AssignTo: &outTESuccess, ReadOnly: true, Text: "",
				Background: SolidColorBrush{Color: walk.RGB(0, 0, 0)},
				TextColor:  walk.RGB(255, 255, 255),
				VScroll:    true,
			},
			TextEdit{
				AssignTo: &outTEErron, ReadOnly: true, Text: "",
				Background: SolidColorBrush{Color: walk.RGB(0, 0, 0)},
				TextColor:  walk.RGB(199, 37, 78),
				VScroll:    true,
			},
		},
	}
}

// RunWindow 启动窗口
func (layout *WindowCustom) RunWindow() {
	_ = MainWindow{
		Title:    "华爱工具箱",
		Size:     Size{Width: 800, Height: 600},
		Layout:   Grid{Columns: 2},
		AssignTo: &windowMain,
		Children: []Widget{
			layout.ToolsLeftDisplay(),
			layout.ToolsRightDisplay(),
		},
	}.Create()
	win.SetWindowLong(windowMain.Handle(), win.GWL_STYLE, win.GetWindowLong(windowMain.Handle(), win.GWL_STYLE) & ^win.WS_MAXIMIZEBOX & ^win.WS_THICKFRAME)
	windowMain.Run()
}
