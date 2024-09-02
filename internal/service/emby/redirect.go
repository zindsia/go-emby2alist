package emby

import (
	"errors"
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/service/alist"
	"go-emby2alist/internal/service/path"
	"go-emby2alist/internal/util/color"
	"go-emby2alist/internal/util/jsons"
	"go-emby2alist/internal/web/cache"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Redirect2AlistLink 重定向资源到 alist 网盘直链
func Redirect2AlistLink(c *gin.Context) {
	// 1 解析要请求的资源信息
	itemInfo, err := resolveItemInfo(c)
	if checkErr(c, err) {
		return
	}
	log.Printf(color.ToBlue("解析到的 itemInfo: %v"), jsons.NewByVal(itemInfo))

	// 2 请求资源在 Emby 中的 Path 参数
	embyPath, err := getEmbyFileLocalPath(itemInfo.PlaybackInfoUri)
	if checkErr(c, err) {
		return
	}

	// 3 请求 alist 资源
	alistPathRes := path.Emby2Alist(embyPath)
	if alistPathRes.Success {
		log.Printf(color.ToBlue("尝试请求 Alist 资源: %s"), alistPathRes.Path)
		res := alist.FetchResource(alistPathRes.Path, itemInfo.MsInfo.Transcode, itemInfo.MsInfo.TemplateId, false)

		if res.Code == http.StatusOK {
			log.Printf(color.ToGreen("请求成功, 重定向到: %s"), res.Data)
			c.Header(cache.HeaderKeyExpired, cache.Duration(time.Minute*10))
			c.Redirect(http.StatusFound, res.Data)
			return
		}

		if res.Code == http.StatusForbidden {
			log.Printf(color.ToRed("请求 Alist 被阻止: %s"), res.Msg)
			c.String(http.StatusForbidden, res.Msg)
		}
	}

	paths, err := alistPathRes.Range()
	if checkErr(c, err) {
		return
	}

	for _, path := range paths {
		log.Printf(color.ToBlue("尝试请求 Alist 资源: %s"), path)
		res := alist.FetchResource(path, itemInfo.MsInfo.Transcode, itemInfo.MsInfo.TemplateId, true)

		if res.Code == http.StatusOK {
			log.Printf(color.ToGreen("请求成功, 重定向到: %s"), res.Data)
			c.Header(cache.HeaderKeyExpired, cache.Duration(time.Minute*10))
			c.Redirect(http.StatusFound, res.Data)
			return
		}
	}

	checkErr(c, errors.New("查无 Alist 直链资源"))
}

// checkErr 检查 err 是否为空
// 不为空则重定向到源服务器
//
// 返回 true 表示已重定向
func checkErr(c *gin.Context, err error) bool {
	if err == nil || c == nil {
		return false
	}
	u := config.C.Emby.Host + c.Request.URL.String()
	log.Printf(color.ToRed("代理接口失败: %v, 重定向回源服务器处理\n"), err)

	// 异常接口, 不缓存
	c.Header(cache.HeaderKeyExpired, "-1")
	c.Redirect(http.StatusTemporaryRedirect, u)
	return true
}
