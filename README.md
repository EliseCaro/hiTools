# 华爱科技工具箱

### 打包应用
``` bash
  0：go get github.com/akavel/rsrc
  1：rsrc -manifest hi.manifest -o rsrc.syso -ico ./favicon.ico
  2：go build -ldflags="-H windowsgui"
```
