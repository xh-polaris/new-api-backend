package middleware

import (
	"net/http"
	"one-api/util"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func JWT2Session(c *gin.Context) {
	if token := c.GetHeader("Authorization"); token != "" {
		split := strings.Replace(token, "Bearer ", "", 1)
		claims, err := util.ParseToken(split)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "无权进行此操作，未提供有效的 Authorization",
			})
			return
		}

		session := sessions.Default(c)
		// 为每个字段设置 session 值
		session.Set("id", claims.ID)
		session.Set("username", claims.Username)
		session.Set("role", claims.Role)
		session.Set("status", claims.Status)
		session.Set("group", claims.Group)
		session.Set("turnstile", claims.Turnstile)
		session.Set("pending_username", claims.PendingUsername)
		session.Set("pending_user_id", claims.PendingUserID)
		session.Set("aff", claims.AffCode)
		session.Set("oauth_state", claims.OAuthState)

		// 保存 session
		if err := session.Save(); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "无法保存 session 数据",
			})
			return
		}
	}
	c.Next()
}
