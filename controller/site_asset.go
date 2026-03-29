package controller

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxSiteAssetSize = 2 * 1024 * 1024

var allowedSiteAssetExtensions = map[string]struct{}{
	".png":  {},
	".jpg":  {},
	".jpeg": {},
	".webp": {},
}

func UploadSiteAsset(c *gin.Context) {
	category := c.PostForm("category")
	if category != "help-group-card" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "不支持的资源分类",
		})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请选择要上传的图片",
		})
		return
	}
	if fileHeader.Size > maxSiteAssetSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "图片大小不能超过 2MB",
		})
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if _, ok := allowedSiteAssetExtensions[ext]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "仅支持 png/jpg/jpeg/webp 图片",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer file.Close()

	sniffBuffer := make([]byte, 512)
	readLength, err := file.Read(sniffBuffer)
	if err != nil && err != io.EOF {
		common.ApiError(c, err)
		return
	}
	mimeType := http.DetectContentType(sniffBuffer[:readLength])
	if !strings.HasPrefix(mimeType, "image/png") &&
		!strings.HasPrefix(mimeType, "image/jpeg") &&
		!strings.HasPrefix(mimeType, "image/webp") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "上传文件不是合法图片",
		})
		return
	}

	targetDir := filepath.Join(common.SiteAssetDir, category)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		common.ApiError(c, err)
		return
	}

	fileName := uuid.NewString() + ext
	filePath := filepath.Join(targetDir, fileName)
	if err := c.SaveUploadedFile(fileHeader, filePath); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"url":       path.Join("/site-assets", category, fileName),
			"path":      filepath.ToSlash(filepath.Join(category, fileName)),
			"mime_type": mimeType,
			"size":      fileHeader.Size,
		},
	})
}

func ServeSiteAsset(c *gin.Context) {
	requestPath := strings.TrimPrefix(c.Param("filepath"), "/")
	if requestPath == "" {
		c.Status(http.StatusNotFound)
		return
	}

	cleanRequestPath := path.Clean("/" + requestPath)
	if cleanRequestPath == "/" {
		c.Status(http.StatusNotFound)
		return
	}
	relativePath := strings.TrimPrefix(cleanRequestPath, "/")

	basePath, err := filepath.Abs(common.SiteAssetDir)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	assetPath, err := filepath.Abs(filepath.Join(basePath, filepath.FromSlash(relativePath)))
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if assetPath != basePath && !strings.HasPrefix(assetPath, basePath+string(os.PathSeparator)) {
		c.Status(http.StatusNotFound)
		return
	}
	if _, err := os.Stat(assetPath); err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.File(assetPath)
}
