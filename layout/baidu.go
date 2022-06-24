package layout

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type BaiduService struct{}

type BaiduCheckValue struct {
	Proxy string // 代理
	Urls  string // 采集地址
	Push  string // 发布地址
}

var BaiduCheckConfig = new(BaiduCheckValue)

/** ****************************************************************窗口部分开始*****************************************************/

// BaiduSdkDialog 百度弹窗
func (service *BaiduService) BaiduSdkDialog(owner *walk.MainWindow, baidu *BaiduCheckValue) (int, error) {
	var dlg *walk.Dialog
	var db *walk.DataBinder
	var acceptPB, cancelPB *walk.PushButton
	return Dialog{
		AssignTo:      &dlg,
		Title:         "百度百家号采集",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		MinSize:       Size{Width: 400},
		Layout:        VBox{},
		DataBinder:    DataBinder{AssignTo: &db, Name: "baidu", DataSource: baidu, ErrorPresenter: ToolTipErrorPresenter{}},
		Children: []Widget{
			Composite{Layout: Grid{Columns: 2}, Children: service.DialogFieldForm()},
			Composite{Layout: HBox{}, Children: []Widget{Composite{Layout: Grid{Columns: 2}, Children: []Widget{PushButton{AssignTo: &acceptPB, Text: "确定", OnClicked: func() { _ = db.Submit(); dlg.Accept() }}, PushButton{AssignTo: &cancelPB, Text: "取消", OnClicked: func() { dlg.Cancel() }}}}}},
		},
	}.Run(owner)
}

// BaiduSdk 百度百家号采集
func (service *BaiduService) BaiduSdk() Composite {
	return Composite{
		Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
		Layout:     HBox{},
		Children: []Widget{
			PushButton{Text: "百家号采集", OnClicked: func() {
				service.BaiduCheckConfigLoad() // 加载本地配置
				if cmd, err := service.BaiduSdkDialog(windowMain, BaiduCheckConfig); err != nil {
					message := fmt.Sprintf(`弹窗失败：失败原因：%s`, err.Error())
					new(WindowCustom).ConsoleErron(message)
				} else if cmd == walk.DlgCmdOK {
					service.BaiduCheckConfigSet()
					service.ReadCvsHandle(BaiduCheckConfig.Urls)
				}
			}},
			PushButton{Text: "开始", OnClicked: func() {
				service.BaiduCheckConfigLoad() // 加载本地配置
				if service.BaiduCheckConfigSet() {
					go service.BaiduCollectStart()
				}
			}, MaxSize: Size{Width: 50}},
		},
	}
}

// DialogField 弹窗字段
func (service *BaiduService) DialogField() map[string]string {
	return map[string]string{
		"Proxy": "小象代理地址",
		"Urls":  "采集队列地址",
		"Push":  "站点发布地址",
	}
}

// DialogFieldForm 弹窗元素对象
func (service *BaiduService) DialogFieldForm() []Widget {
	var display []Widget
	for index, val := range service.DialogField() {
		display = append(display, Label{Text: fmt.Sprintf(`%s：`, val)}, LineEdit{Text: Bind(index)})
	}
	return display
}

// BaiduCheckConfigSet 生成配置文件
func (service *BaiduService) BaiduCheckConfigSet() bool {
	var context string
	var maps map[string]string
	bs, _ := filepath.Abs("config")
	mjson, _ := json.Marshal(BaiduCheckConfig)
	_ = json.Unmarshal(mjson, &maps)
	for index, val := range service.DialogField() {
		if _, ok := maps[index]; ok == false || len([]rune(maps[index])) <= 0 {
			new(WindowCustom).ConsoleErron(fmt.Sprintf(`配置错误；请正确配置[%s]的值;`, val))
			return false
		} else {
			text := strings.Replace(maps[index], ":", "#", -1)
			context += fmt.Sprintf("%s:%s\r\n", index, text)
		}
	}
	config := fmt.Sprintf(`%s\baidu.config`, bs)
	return new(RequestService).CreateConfig(config, context)
}

// BaiduCheckConfigLoad 初始化加载本地配置
func (service *BaiduService) BaiduCheckConfigLoad() {
	bs, _ := filepath.Abs("config")
	config := fmt.Sprintf(`%s\baidu.config`, bs)
	cgf := new(RequestService).TextConfig(config)
	mjson, _ := json.Marshal(cgf)
	mjson = []byte(strings.Replace(string(mjson), "#", ":", -1))
	_ = json.Unmarshal(mjson, &BaiduCheckConfig)
}

// ReadCvsHandle 将csv文件进行格式组装
func (service *BaiduService) ReadCvsHandle(filename string) {
	//bs, _ := filepath.Abs("config")
	//config := fmt.Sprintf(`%s\baidu.txt`, bs)
	if fs, err := os.Open(filename); err != nil {
		message := fmt.Sprintf(`文件[%s]解析失败，%s`, filename, err.Error())
		new(WindowCustom).ConsoleErron(message)
	} else {
		var text string
		defer func(fs *os.File) { _ = fs.Close() }(fs)
		r := csv.NewReader(fs)
		for {
			row, err := r.Read()
			if err != nil && err != io.EOF {
				new(WindowCustom).ConsoleErron(fmt.Sprintf(`文件[%s]读取行失败，%s`, filename, err.Error()))
			}
			if err == io.EOF {
				break
			}
			if len(row) == 2 && row[1] != "" && row[0] != "" {
				text += fmt.Sprintf("%s&collect=%s\r\n", service.GbkToUtf8([]byte(row[1])), service.GbkToUtf8([]byte(row[0])))
			}
		}
		fmt.Println(text)
	}

}

/** ****************************************************************窗口部分结束*****************************************************/
/** ****************************************************************以下开始采集部分*****************************************************/

func (service *BaiduService) Collect() {

}

func (service *BaiduService) BaiduCollectStart() {
	if url := service.GetFileUrls(); url == "" {
		new(WindowCustom).ConsoleErron("获取不到任何待采集Url；请检查队列是否存在;")
	} else {
		fmt.Println(url)
	}
}

// GetFileUrls 从文件中获取一个url
func (service *BaiduService) GetFileUrls() string {
	if urls := service.ReadTxt(BaiduCheckConfig.Urls); len(urls) <= 0 {
		return ""
	} else {
		return urls[len(urls)-1]
	}
}

// ReadTxt 读取text
func (service *BaiduService) ReadTxt(local string) []string {
	var items []string
	if file, erron := os.Open(local); erron == nil {
		reader := bufio.NewReader(file)
		for {
			if line, _, err := reader.ReadLine(); err != nil || err == io.EOF {
				break
			} else {
				items = append(items, strings.Trim(string(line), " "))
			}
		}
		defer func(f *os.File) { _ = f.Close() }(file)
	}
	return items
}

// GbkToUtf8 编码转换
func (service *BaiduService) GbkToUtf8(s []byte) string {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, _ := ioutil.ReadAll(reader)
	return string(d)
}
