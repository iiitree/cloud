package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// 简单的身份验证中间件
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, hasAuth := c.Request.BasicAuth()
		if !hasAuth || username != "admin" || password != "password" {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

func main() {
	router := gin.Default()

	// 首页路由
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, FileBrowser!")
	})

	// 文件上传路由
	router.POST("/upload", func(c *gin.Context) {
		// 获取表单中的文件
		file, err := c.FormFile("file")
		if err != nil {
			c.String(http.StatusBadRequest, "文件上传失败: %v", err)
			return
		}

		// 指定保存文件的路径
		savePath := filepath.Join("./uploads", file.Filename)

		// 创建保存文件的目录
		if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
			c.String(http.StatusInternalServerError, "无法创建目录: %v", err)
			return
		}

		// 保存文件
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.String(http.StatusInternalServerError, "无法保存文件: %v", err)
			return
		}

		c.String(http.StatusOK, fmt.Sprintf("'%s' 上传成功!", file.Filename))
	})

	// 文件下载路由
	router.GET("/download/:filename", func(c *gin.Context) {
		// 获取文件名参数
		filename := c.Param("filename")
		filePath := filepath.Join("./uploads", filename)

		// 检查文件是否存在
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			c.String(http.StatusNotFound, "文件不存在: %v", err)
			return
		}

		// 提供文件下载
		c.File(filePath)
	})

	// 文件列表路由
	router.GET("/files", func(c *gin.Context) {
		files := []string{}

		// 读取目录下的文件列表
		err := filepath.Walk("./uploads", func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				files = append(files, info.Name())
			}
			return nil
		})

		if err != nil {
			c.String(http.StatusInternalServerError, "无法读取文件列表: %v", err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"files": files})
	})

	// 文件删除路由
	router.DELETE("/delete/:filename", func(c *gin.Context) {
		// 获取文件名参数
		filename := c.Param("filename")
		filePath := filepath.Join("./uploads", filename)

		// 检查文件是否存在
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			c.String(http.StatusNotFound, "文件不存在: %v", err)
			return
		}

		// 删除文件
		if err := os.Remove(filePath); err != nil {
			c.String(http.StatusInternalServerError, "无法删除文件: %v", err)
			return
		}

		c.String(http.StatusOK, fmt.Sprintf("文件 '%s' 已成功删除", filename))
	})

	// 目录浏览路由，返回美化后的 HTML 页面
	router.GET("/browse/*filepath", func(c *gin.Context) {
		root := "./uploads"
		requestedPath := c.Param("filepath")
		fullPath := filepath.Join(root, requestedPath)

		fileInfo, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			c.String(http.StatusNotFound, "路径不存在: %v", err)
			return
		}

		// 检查请求的路径是否是一个目录
		if !fileInfo.IsDir() {
			c.String(http.StatusBadRequest, "请求的路径不是一个目录")
			return
		}

		files := []os.FileInfo{}
		dir, err := os.Open(fullPath)
		if err != nil {
			c.String(http.StatusInternalServerError, "无法打开目录: %v", err)
			return
		}
		defer dir.Close()

		files, err = dir.Readdir(-1)
		if err != nil {
			c.String(http.StatusInternalServerError, "无法读取目录内容: %v", err)
			return
		}

		// 生成带有 Bootstrap 样式的 HTML 页面
		htmlContent := `<html>
    <head>
        <meta charset='UTF-8'>
        <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.5.2/css/bootstrap.min.css">
    </head>
    <body>
        <div class="container">
            <h1 class="mt-4">目录浏览: ` + requestedPath + `</h1>
            <ul class="list-group mt-3">`

		// 添加返回上一级目录的链接
		if requestedPath != "/" {
			parentPath := filepath.Dir(requestedPath)
			if parentPath == "." {
				parentPath = "/"
			}
			htmlContent += fmt.Sprintf(`<li class="list-group-item"><a href='/browse%s'>.. (上一级目录)</a></li>`, parentPath)
		}

		for _, file := range files {
			name := file.Name()
			link := filepath.Join(requestedPath, name)

			if file.IsDir() {
				htmlContent += fmt.Sprintf(`<li class="list-group-item"><a href='/browse%s'>%s/</a></li>`, link, name)
			} else {
				htmlContent += fmt.Sprintf(`<li class="list-group-item"><a href='/download%s'>%s</a></li>`, link, name)
			}
		}

		htmlContent += `</ul>
        </div>
    </body>
    </html>`

		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, htmlContent)
	})

	// 启动服务器
	router.Run(":8080")
}
