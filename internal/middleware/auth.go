package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/agriconnect-ai/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const ContextKeyUser = "auth_user"

type AuthUser struct {
	ID                uuid.UUID
	FullName          string
	PhoneNumber       string
	District          string
	PreferredLanguage string
	Role              string
	IsActive          bool
}

func wantsHTML(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	return strings.Contains(accept, "text/html")
}

func AuthRequired(accessSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := extractUser(c, accessSecret)
		if err != nil {
			if wantsHTML(c) {
				c.Redirect(http.StatusSeeOther, "/login")
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			}
			c.Abort()
			return
		}
		c.Set(ContextKeyUser, user)
		c.Set("user_id", user.ID.String())
		c.Set("user_role", user.Role)
		c.Next()
	}
}

func OptionalAuth(accessSecret string, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := extractUser(c, accessSecret)
		if err == nil && user != nil {
			c.Set(ContextKeyUser, user)
			c.Set("user_id", user.ID.String())
			c.Set("user_role", user.Role)
		}
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRaw, exists := c.Get(ContextKeyUser)
		if !exists {
			if wantsHTML(c) {
				c.Redirect(http.StatusSeeOther, "/login")
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			}
			c.Abort()
			return
		}
		user, ok := userRaw.(*AuthUser)
		if !ok {
			if wantsHTML(c) {
				c.Redirect(http.StatusSeeOther, "/login")
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			}
			c.Abort()
			return
		}

		for _, role := range roles {
			if user.Role == role {
				c.Next()
				return
			}
		}

		if wantsHTML(c) {
			c.HTML(http.StatusForbidden, "error.html", gin.H{
				"ErrorCode":    403,
				"ErrorMessage": "Insufficient permissions. You don't have access to this page.",
			})
		} else {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		}
		c.Abort()
	}
}

func extractUser(c *gin.Context, accessSecret string) (*AuthUser, error) {
	accessTokenStr, err := c.Cookie("access_token")
	if err != nil || accessTokenStr == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			accessTokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if accessTokenStr == "" {
		return nil, errors.New("no token")
	}

	claims, err := auth.ValidateToken(accessTokenStr, accessSecret)
	if err != nil {
		return nil, err
	}

	return &AuthUser{
		ID:   claims.UserID,
		Role: claims.Role,
	}, nil
}
