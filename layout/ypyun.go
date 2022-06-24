package layout

import (
	"encoding/json"
	"fmt"
	"github.com/EliseCaro/go_image"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/upyun/go-sdk/v3/upyun"
	"hiTools/layout/job"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type YpYunService struct{}

// CropImageValue 又拍云配置结构
type CropImageValue struct {
	Directory string
	Bucket    string
	Operator  string
	Password  string
	Domain    string
	Exif      string
	Width     string
	Height    string
}

// CropImageConfig 又拍云配置值
var CropImageConfig = new(CropImageValue)

// CropImageReqStart 开始处理入口
func (service *YpYunService) CropImageReqStart() {
	service.ListFiles()
}

// ListFiles 获取文件列表
func (service *YpYunService) ListFiles() {
	objsChan := make(chan *upyun.FileInfo, 10)
	go func() {
		erron := service.initSdk().List(
			&upyun.GetObjectsConfig{
				Path:        CropImageConfig.Directory,
				ObjectsChan: objsChan,
			})
		if erron != nil {
			new(WindowCustom).ConsoleErron(fmt.Sprintf(`链接到又拍云失败;原因:%s`, erron.Error()))
		}
	}()
	poolService := JobPoolService{}
	for obj := range objsChan {
		poolService.GetPool().PushTaskFunc(func(w *job.Pool, args ...interface{}) job.Flag {
			service.CheckFileSize(args[0].(string))
			return job.FLAG_OK
		}, obj.Name)
	}
	new(WindowCustom).ConsoleSuccess(fmt.Sprintf(`[%s]空间[%s]资源处理完成;`, CropImageConfig.Bucket, CropImageConfig.Directory))
}

// initSdk 实例化SDK
func (service *YpYunService) initSdk() *upyun.UpYun {
	return upyun.NewUpYun(&upyun.UpYunConfig{
		Bucket:   CropImageConfig.Bucket,
		Operator: CropImageConfig.Operator,
		Password: CropImageConfig.Password,
	})
}

// CheckFileSize 检测大小是否处理
func (service *YpYunService) CheckFileSize(name string) {
	urls := fmt.Sprintf(`%s%s/%s`, CropImageConfig.Domain, CropImageConfig.Directory, name)
	resp, erron := new(RequestService).GetReq(fmt.Sprintf(`%s!%s`, urls, CropImageConfig.Exif))
	if erron == nil && len(resp) > 0 {
		var people map[string]interface{}
		if erron = json.Unmarshal(resp, &people); erron != nil {
			new(WindowCustom).ConsoleErron(fmt.Sprintf(`图片信息转换失败;原因:%s`, erron.Error()))
		} else {
			if _, ok := people["width"]; ok {
				width, _ := strconv.Atoi(CropImageConfig.Width)
				height, _ := strconv.Atoi(CropImageConfig.Height)
				if int(people["width"].(float64)) > width || int(people["height"].(float64)) > height {
					new(WindowCustom).ConsoleSuccess(fmt.Sprintf(`[%s]处理开始`, name))
					service.download(name)
				} else {
					new(WindowCustom).ConsoleSuccess(fmt.Sprintf(`[%s]尺寸小于或等于预设值，将跳过~`, name))
				}
			} else {
				new(WindowCustom).ConsoleErron("请求图片信息失败;请确认图片信息版本是否配置正确！")
			}
		}
	} else {
		new(WindowCustom).ConsoleErron(fmt.Sprintf(`请求图片信息失败;原因:%s`, erron.Error()))
	}
}

// CheckFileSize 检测大小是否处理
func (service *YpYunService) download(name string) {
	lcpat, _ := filepath.Abs("image")
	_, erron := service.initSdk().Get(&upyun.GetObjectConfig{
		Path:      fmt.Sprintf(`%s/%s`, CropImageConfig.Directory, name),
		LocalPath: fmt.Sprintf(`%s\%s`, lcpat, name),
	})
	if erron == nil {
		local := fmt.Sprintf(`%s\%s`, lcpat, name)
		width, _ := strconv.Atoi(CropImageConfig.Width)
		height, _ := strconv.Atoi(CropImageConfig.Height)
		if erron = go_image.ThumbnailF2F(local, local, width, height); erron == nil {
			if erron = service.initSdk().Put(&upyun.PutObjectConfig{
				Path:      fmt.Sprintf(`%s/%s`, CropImageConfig.Directory, name),
				LocalPath: local,
			}); erron == nil {
				new(WindowCustom).ConsoleSuccess(fmt.Sprintf(`[%s]处理成功~`, name))
			} else {
				new(WindowCustom).ConsoleErron(fmt.Sprintf(`[%s]上传失败;原因:%s`, name, erron.Error()))
			}
			_ = os.Remove(local)
		} else {
			new(WindowCustom).ConsoleErron(fmt.Sprintf(`[%s]剪切失败;原因:%s`, name, erron.Error()))
		}
	} else {
		new(WindowCustom).ConsoleErron(fmt.Sprintf(`[%s]下载失败;原因:%s`, name, erron.Error()))
	}
}

// CropImageDialog 又拍云配置弹窗
func (service *YpYunService) CropImageDialog(owner *walk.MainWindow, animal *CropImageValue) (int, error) {
	var dlg *walk.Dialog
	var db *walk.DataBinder
	var acceptPB, cancelPB *walk.PushButton
	return Dialog{
		AssignTo:      &dlg,
		Title:         "又拍云空间配置",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		MinSize:       Size{Width: 400},
		Layout:        VBox{},
		DataBinder:    DataBinder{AssignTo: &db, Name: "animal", DataSource: animal, ErrorPresenter: ToolTipErrorPresenter{}},
		Children: []Widget{
			Composite{Layout: Grid{Columns: 2}, Children: service.DialogFieldForm()},
			Composite{Layout: HBox{}, Children: []Widget{Composite{Layout: Grid{Columns: 2}, Children: []Widget{PushButton{AssignTo: &acceptPB, Text: "确定", OnClicked: func() { _ = db.Submit(); dlg.Accept() }}, PushButton{AssignTo: &cancelPB, Text: "取消", OnClicked: func() { dlg.Cancel() }}}}}},
		},
	}.Run(owner)
}

func (service *YpYunService) CropImageConfigLoad() {
	bs, _ := filepath.Abs("config")
	config := fmt.Sprintf(`%s\ypyun-crop.config`, bs)
	cgf := new(RequestService).TextConfig(config)
	mjson, _ := json.Marshal(cgf)
	mjson = []byte(strings.Replace(string(mjson), "#", ":", -1))
	_ = json.Unmarshal(mjson, &CropImageConfig)
}

func (service *YpYunService) DialogField() map[string]string {
	return map[string]string{
		"Operator":  "操作员",
		"Height":    "图片高",
		"Width":     "图片宽",
		"Bucket":    "操作空间",
		"Domain":    "资源域名",
		"Password":  "操作员密码",
		"Exif":      "图片版本标识",
		"Directory": "资源处理目录",
	}
}

func (service *YpYunService) DialogFieldForm() []Widget {
	var display []Widget
	for index, val := range service.DialogField() {
		display = append(display, Label{Text: fmt.Sprintf(`%s：`, val)}, LineEdit{Text: Bind(index)})
	}
	return display
}

func (service *YpYunService) CropImageConfigSet() bool {
	var context string
	var maps map[string]string
	bs, _ := filepath.Abs("config")
	mjson, _ := json.Marshal(CropImageConfig)
	_ = json.Unmarshal(mjson, &maps)
	for index, val := range service.DialogField() {
		if _, ok := maps[index]; ok == false || len([]rune(maps[index])) <= 0 {
			new(WindowCustom).ConsoleErron(fmt.Sprintf(`配置错误；请正确配置[%s]的值;`, val))
			return false
		} else {
			context += fmt.Sprintf("%s:%s\r\n", index, strings.Replace(maps[index], ":", "#", -1))
		}
	}
	config := fmt.Sprintf(`%s\ypyun-crop.config`, bs)
	return new(RequestService).CreateConfig(config, context)
}

// CropImage 又拍云图片剪切功能
func (service *YpYunService) CropImage() Composite {
	return Composite{
		Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
		Layout:     HBox{},
		Children: []Widget{
			PushButton{Text: "批量剪切", OnClicked: func() {
				new(YpYunService).CropImageConfigLoad() // 加载本地配置
				if cmd, err := new(YpYunService).CropImageDialog(windowMain, CropImageConfig); err != nil {
					new(WindowCustom).ConsoleErron(fmt.Sprintf(`弹窗失败：失败原因：%s`, err.Error()))
				} else if cmd == walk.DlgCmdOK {
					new(YpYunService).CropImageConfigSet()
				}
			}},
			PushButton{Text: "开始", OnClicked: func() {
				new(YpYunService).CropImageConfigLoad() // 加载本地配置
				if new(YpYunService).CropImageConfigSet() {
					go new(YpYunService).CropImageReqStart()
				}
			}, MaxSize: Size{Width: 50}},
		},
	}
}
