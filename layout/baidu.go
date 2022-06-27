package layout

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	imgext "github.com/shamsher31/goimgext"
	"github.com/upyun/go-sdk/v3/upyun"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type BaiduService struct{}

type BaiduCheckValue struct {
	Proxy      string // 代理
	Urls       string // 采集地址
	Bucket     string //  又拍云空间名
	Operator   string // 又拍云操作员
	Password   string // 又拍云操作密码
	DomainUrls string // 又拍云
	PushId     string // 发布分类ID
	PushUrl    string // 发布推送地址
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
		"Proxy":      "小象代理地址",
		"Urls":       "采集队列地址",
		"Bucket":     "又拍云空间名",
		"Operator":   "又拍云操作员",
		"Password":   "又拍云操作密码",
		"DomainUrls": "又拍云链接域名",
		"PushId":     "发布分类ID",
		"PushUrl":    "发布推送地址",
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
func (service *BaiduService) ReadCvsHandle(filename string) bool {
	bs, _ := filepath.Abs("config")
	config := fmt.Sprintf(`%s\baidu.txt`, bs)
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
		return new(RequestService).CreateConfig(config, text)
	}
	return false
}

/** ****************************************************************窗口部分结束*****************************************************/
/** ****************************************************************以下开始采集部分*****************************************************/

func (service *BaiduService) Collect(urls string, proxy bool) {
	proxyUrls := BaiduCheckConfig.Proxy
	request := new(RequestService).RespHtmlBaidu(urls)
	body := request.requestCustom(proxy, proxyUrls)
	if len(body) == 0 || string(body) == "" {
		new(WindowCustom).ConsoleErron(fmt.Sprintf(`URL[%s]采集失败，换IP重试中`, urls))
		service.Collect(urls, true)
	}
	ios := strings.NewReader(string(body))
	if dom, err := goquery.NewDocumentFromReader(ios); err != nil {
		new(WindowCustom).ConsoleErron(fmt.Sprintf(`URL[%s]解析Html失败，换IP重试中`, urls))
		service.Collect(urls, true)
	} else {
		if context := dom.Find(".index-module_articleWrap_2Zphx "); context == nil {
			new(WindowCustom).ConsoleErron(fmt.Sprintf(`URL[%s]触发百度验证，换IP重试中`, urls))
		} else {
			content, _ := context.Html()
			content = service.filtrationHtml(content)
			content = service.imageHandle(content)
			kwd := strings.Replace(urls[strings.Index(urls, "&collect="):len(urls)], "&collect=", "", -1)
			service.UpdateTxt(urls)
			if len([]rune(service.TrimHtml(content))) <= 50 || len([]rune(service.TrimHtml(kwd))) <= 0 {
				new(WindowCustom).ConsoleErron(fmt.Sprintf(`关键词[%s]解析内容或关键词为空，移除关键词，终止发布~`, kwd))
			}
			service.PushData(content, kwd) //
			service.BaiduCollectStart()    // 开始下一轮
		}
	}
}

// TrimHtml 剔除html标签
func (service *BaiduService) TrimHtml(src string) string {
	re, _ := regexp.Compile("\\<[\\S\\s]+?\\>")
	src = re.ReplaceAllStringFunc(src, strings.ToLower)
	re, _ = regexp.Compile("\\<style[\\S\\s]+?\\</style\\>")
	src = re.ReplaceAllString(src, "")
	re, _ = regexp.Compile("\\<script[\\S\\s]+?\\</script\\>")
	src = re.ReplaceAllString(src, "")
	re, _ = regexp.Compile("\\<[\\S\\s]+?\\>")
	src = re.ReplaceAllString(src, "\n")
	src = strings.Replace(src, "\n", "", -1)
	re, _ = regexp.Compile("\\{!--[\\S\\s]+?\\--}")
	src = re.ReplaceAllString(src, "\n")
	return strings.TrimSpace(src)
}

// PushData 发布文章
// 发布地址：/post.php?action=save&secret=发布密码
// post:{post_title:"文章标题","post_content":文章内容,"tag":"标签","post_category":"分类id"}
func (service *BaiduService) PushData(context, kwd string) {
	if erron := service.PushRequest(context, kwd); erron != nil {
		new(WindowCustom).ConsoleErron(erron.Error())
	} else {
		new(WindowCustom).ConsoleSuccess(fmt.Sprintf(`关键词[%s]采集成功，发布成功~`, kwd))
	}
}

// PushRequest 启动post发布
func (service *BaiduService) PushRequest(context, kwd string) error {
	body := url.Values{
		"post_title":    {kwd},
		"post_content":  {context},
		"tag":           {kwd},
		"post_category": {BaiduCheckConfig.PushId},
	}
	resp, err := http.PostForm(BaiduCheckConfig.PushUrl, body)
	if err != nil {
		return errors.New(fmt.Sprintf(`发布发起请求失败；%s`, err.Error()))
	}
	defer func(Body io.ReadCloser) { _ = Body.Close() }(resp.Body)
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New(fmt.Sprintf(`发布写入IO失败；%s`, err.Error()))
	}
	return nil
}

// UpdateTxt 更新采集库文件
func (service *BaiduService) UpdateTxt(urls string) {
	bs, _ := filepath.Abs("config")
	config := fmt.Sprintf(`%s\baidu.txt`, bs)
	data, _ := ioutil.ReadFile(config) // 读取文件
	txtCon := strings.Replace(string(data), fmt.Sprintf("%s\r\n", urls), "", -1)
	new(RequestService).CreateConfig(config, txtCon)
}

// imageHandle 图片处理
func (service *BaiduService) imageHandle(html string) string {
	dom, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	dom.Find("img").Each(func(i int, selection *goquery.Selection) {
		if src, _ := selection.Attr("src"); src == "" {
			selection.Remove()
		} else {
			if newsSrc := service.dowImage(src); newsSrc == "" {
				selection.Remove()
			} else {
				selection.RemoveAttr("class")
				selection.RemoveAttr("width")
				selection.RemoveAttr("height")
				selection.SetAttr("src", newsSrc)
			}
		}
	})
	dom.Find("video").Each(func(i int, selection *goquery.Selection) {
		selection.Remove()
	})
	context, _ := dom.Find("body").Html()
	return context
}

// filtrationHtml 过滤内容
func (service *BaiduService) filtrationHtml(html string) string {
	reg, _ := regexp.Compile("<div(.*?)>")
	html = reg.ReplaceAllString(html, "<div>")
	html = strings.Replace(html, "<p></p>", "", -1)
	reg2, _ := regexp.Compile("<p(.*?)>")
	return reg2.ReplaceAllString(html, "<p>")
}

// BaiduCollectStart 启动采集器
func (service *BaiduService) BaiduCollectStart() {
	if url := service.GetFileUrls(); url == "" {
		new(WindowCustom).ConsoleErron("获取不到任何待采集Url；请检查队列是否存在;")
	} else {
		service.Collect(url, false)
	}
}

// GetFileUrls 从文件中获取一个url
func (service *BaiduService) GetFileUrls() string {
	bs, _ := filepath.Abs("config")
	config := fmt.Sprintf(`%s\baidu.txt`, bs)
	if urls := service.ReadTxt(config); len(urls) <= 0 {
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

/** ****************************************************************结束采集部分*****************************************************/

/** ****************************************************************开始下载图片*****************************************************/

func (service *BaiduService) dowImage(src string) string {
	var strlocal string
	if resp, erron := http.Get(src); erron != nil {
		new(WindowCustom).ConsoleErron(fmt.Sprintf(`图片[%s]下载失败，移除元素~`, src))
	} else {
		defer func(Body io.ReadCloser) { _ = Body.Close() }(resp.Body)
		bs, _ := filepath.Abs(`static`)
		local := fmt.Sprintf(`%s/temp-%s`, bs, service.md5V(src))
		out, err := os.Create(local)
		if err != nil {
			new(WindowCustom).ConsoleErron(fmt.Sprintf(`图片[%s]创建画布失败，移除元素~`, src))
		} else {
			if _, err = io.Copy(out, resp.Body); err != nil {
				func(out *os.File) { _ = out.Close() }(out)
				new(WindowCustom).ConsoleErron(fmt.Sprintf(`图片[%s]创建画布失败，移除元素~`, src))
			} else {
				func(out *os.File) { _ = out.Close() }(out)
				strlocal = service.checkImage(local)
			}
		}
	}
	return strlocal
}

// md5V 生成文件名
func (service *BaiduService) md5V(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return fmt.Sprintf(`%s.png`, hex.EncodeToString(h.Sum(nil)))
}

// 检测图片合法性
func (service *BaiduService) checkImage(loc string) string {
	if _, erron := service.GetExt(loc); erron != nil {
		_ = os.Remove(loc)
		new(WindowCustom).ConsoleErron(fmt.Sprintf(`文件[%s]效应失败;%s`, loc, erron.Error()))
		return ""
	} else {
		yplocal := fmt.Sprintf(`/collect/%s`, strings.Replace(path.Base(loc), "temp-", "", -1))
		erron := service.UpyunSdk().Put(&upyun.PutObjectConfig{
			Path:      yplocal,
			LocalPath: loc,
		})
		if erron != nil {
			new(WindowCustom).ConsoleErron(fmt.Sprintf(`文件[%s]上传又拍云失败;%s`, loc, erron.Error()))
			return ""
		}
		_ = os.Remove(loc)
		return fmt.Sprintf(`%s%s`, BaiduCheckConfig.DomainUrls, yplocal)
	}
}

// UpyunSdk 获取又拍云实例
func (service *BaiduService) UpyunSdk() *upyun.UpYun {
	return upyun.NewUpYun(&upyun.UpYunConfig{
		Bucket:   BaiduCheckConfig.Bucket,
		Operator: BaiduCheckConfig.Operator,
		Password: BaiduCheckConfig.Password,
	})
}

//GetExt  检测图片类型
func (service *BaiduService) GetExt(p string) (string, error) {
	file, err := os.Open(p)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer func(file *os.File) { _ = file.Close() }(file)
	buff := make([]byte, 512)
	_, err = file.Read(buff)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	filetype := http.DetectContentType(buff)
	ext := imgext.Get()
	for i := 0; i < len(ext); i++ {
		if strings.Contains(ext[i], filetype[6:len(filetype)]) {
			return filetype, nil
		}
	}
	return "", errors.New("invalid image type")
}
