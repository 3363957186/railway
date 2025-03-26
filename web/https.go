package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func StartNgork() {
	r := gin.Default()

	// 定义一个 GET 请求接口
	r.GET("/test", func(c *gin.Context) {
		// 返回一个字符串表示成功连接
		c.String(http.StatusOK, "连接成功")
	})

	// 启动 HTTPS 服务
	// 注意：替换 cert.pem 和 key.pem 为你自己的证书文件路径
	r.RunTLS(":443", "cert.pem", "huaweiyunwestlaketest.cn_server.key")
}
