package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
)

func GetInvitationUsers(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Vary", "Cookie, Authorization")
	id := c.GetInt("id")
	if id <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false})
		return
	}
	page, size := 1, 20
	var err error
	if value := c.Query("p"); value != "" {
		page, err = strconv.Atoi(value)
		if err != nil {
			c.JSON(400, gin.H{"success": false, "message": "无效的页码"})
			return
		}
	}
	if value := c.Query("page_size"); value != "" {
		size, err = strconv.Atoi(value)
		if err != nil {
			c.JSON(400, gin.H{"success": false, "message": "无效的分页大小"})
			return
		}
	}
	data, err := model.GetInvitationPage(id, page, size, c.Query("keyword"), model.CanViewInvitationAmounts(id))
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": "无法读取邀请名单"})
		return
	}
	if data.CanViewAmounts && !model.CanViewInvitationAmounts(id) {
		data.CanViewAmounts = false
		for i := range data.Items {
			data.Items[i].Amounts = nil
		}
	}
	common.ApiSuccess(c, data)
}
func GetInvitationUserChoices(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	if c.GetInt("role") < common.RoleRootUser {
		c.JSON(403, gin.H{"success": false})
		return
	}
	ids := []int{}
	if raw := c.Query("ids"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			id, err := strconv.Atoi(part)
			if err != nil || id <= 0 {
				c.JSON(400, gin.H{"success": false, "message": "无效的用户ID"})
				return
			}
			ids = append(ids, id)
		}
	}
	choices, err := model.GetInvitationUserChoices(c.Query("keyword"), ids)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": "无法搜索用户"})
		return
	}
	common.ApiSuccess(c, choices)
}
