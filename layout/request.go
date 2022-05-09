package layout

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

type RequestService struct {
	Req *http.Request
}

// TextConfig 读取文件text
func (service *RequestService) TextConfig(local string) map[string]string {
	var context string
	if file, erron := os.Open(local); erron == nil {
		if info, err := file.Stat(); err == nil {
			buffer := make([]byte, info.Size())
			if _, err = file.Read(buffer); err == nil {
				context = string(buffer)
			}
		}
	}
	return service.Config2Maps(context)
}

// Config2Maps 将配置项转为map返回
func (service *RequestService) Config2Maps(str string) map[string]string {
	maps := make(map[string]string)
	spaceRe, _ := regexp.Compile("[,;\\r\\n]+")
	res := spaceRe.Split(strings.Trim(str, ",;\r\n"), -1)
	if strings.Index(str, ":") > 0 {
		for _, v := range res {
			if countSplit := strings.Split(v, ":"); len(countSplit) == 2 {
				maps[countSplit[0]] = countSplit[1]
			}
		}
	}
	return maps
}

// CreateConfig 创建配置文件
func (service *RequestService) CreateConfig(name string, context string) bool {
	f, err := os.Create(name)
	if err == nil {
		defer func(f *os.File) { _ = f.Close() }(f)
		if _, err = f.Write([]byte(context)); err == nil {
			return true
		} else {
			new(WindowCustom).ConsoleErron(fmt.Sprintf(`写入配置文件失败：原因：%s`, err.Error()))
		}
	}
	new(WindowCustom).ConsoleErron(fmt.Sprintf(`创建配置文件失败：原因：%s`, err.Error()))
	return false
}

// GetReq Get请求 用于又拍云获取图片信息
func (service *RequestService) GetReq(urls string) ([]byte, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", urls, nil)
	if err != nil {
		return []byte{}, err
	}
	request.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	request.Header.Add("Accept-Language", "ja,zh-CN;q=0.8,zh;q=0.6")
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:12.0) Gecko/20100101 Firefox/12.0")
	response, err := client.Do(request)
	defer func(Body io.ReadCloser) { _ = Body.Close() }(response.Body)
	body, _ := ioutil.ReadAll(response.Body)
	return body, nil
}

// 基本自定义请求库公用
func (service *RequestService) requestCustom(proxy bool, proxyUrl string) (body []byte) { // 自定义请求
	if httpClient := service.proxyGetIp(proxy, proxyUrl); httpClient != nil {
		service.Req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.54 Safari/537.36")
		resp, err := httpClient.Do(service.Req)
		if err != nil || resp.Body == nil {
			return service.requestCustom(true, proxyUrl)
		}
		body, _ = ioutil.ReadAll(resp.Body)
		defer func(Body io.ReadCloser) { _ = Body.Close() }(resp.Body)
		return
	} else {
		time.Sleep(10 * time.Second) // 获取代理错误等待10s
		return service.requestCustom(true, proxyUrl)
	}
}

// RespHtmlIcp 组装备案查询请求头
func (service *RequestService) RespHtmlIcp(urls string) *RequestService {
	service.Req, _ = http.NewRequest("GET", urls, nil)
	return service
}

// RespOfficial 组装官网标志查询请求头
func (service *RequestService) RespOfficial(urls, doamin string) *RequestService {
	service.Req, _ = http.NewRequest("GET", urls, nil)
	q := service.Req.URL.Query()
	q.Add("ie", "UTF-8")
	q.Add("wd", fmt.Sprintf(`site:%s`, doamin))
	service.Req.URL.RawQuery = q.Encode()
	service.Req.Header.Add("Host", "www.baidu.com")
	service.Req.Header.Add("Referer", service.Req.URL.String())
	return service
}

// RespHtmlSeo 组装权重查询请求头
func (service *RequestService) RespHtmlSeo(urls, domain string) *RequestService {
	service.Req, _ = http.NewRequest("GET", urls, nil)
	service.Req.Header.Add("Host", "rank.chinaz.com")
	service.Req.Header.Add("Referer", fmt.Sprintf("https://rank.chinaz.com/%s", domain))
	return service
}

func (service *RequestService) TrimHtml(src string) string {
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

func (service *RequestService) proxyGetIp(proxy bool, proxyUrl string) *http.Client {
	if proxy == true || ProxyCore == "" || len(ProxyCore) == 0 {
		if len([]rune(proxyUrl)) <= 0 || proxyUrl == "" {
			new(WindowCustom).ConsoleErron("请配置代理后再试！")
			return nil
		} else {
			if resp, err := http.Get(proxyUrl); err != nil {
				return nil
			} else {
				body, _ := ioutil.ReadAll(resp.Body)
				defer func(Body io.ReadCloser) { _ = Body.Close() }(resp.Body)
				return service.processProxy(body)
			}
		}
	} else {
		return service.processProxy([]byte(ProxyCore))
	}
}

func (service *RequestService) processProxy(body []byte) *http.Client {
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	if _, ok := resp["code"]; ok == false || resp["code"].(float64) != 200 {
		ProxyCore = ""
		new(WindowCustom).ConsoleErron(fmt.Sprintf("获取代理失败；原因：%s", resp["msg"].(string)))
		return nil
	}
	data := resp["data"].([]interface{})[0].(map[string]interface{})
	urls := fmt.Sprintf(`http://%s:%d/`, data["ip"].(string), int(data["port"].(float64)))
	proxy, _ := url.Parse(urls)
	httpClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxy)}}
	ProxyCore = string(body)
	return httpClient
}
