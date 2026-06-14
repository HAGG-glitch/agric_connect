package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const CookieName = "agriconnect_user"
const CookieMaxAge = 60 * 60 * 24 * 365

func AnonymousUser(secure bool, domain string, sameSite string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(CookieName)
		if err != nil || cookie == "" {
			newID := uuid.New().String()
			ss := http.SameSiteLaxMode
			if sameSite == "strict" {
				ss = http.SameSiteStrictMode
			} else if sameSite == "none" {
				ss = http.SameSiteNoneMode
			}
			c.SetSameSite(ss)
			c.SetCookie(CookieName, newID, CookieMaxAge, "/", domain, secure, true)
			c.Set("user_id", newID)
			c.Next()
			return
		}

		if _, err := uuid.Parse(cookie); err != nil {
			newID := uuid.New().String()
			ss := http.SameSiteLaxMode
			if sameSite == "strict" {
				ss = http.SameSiteStrictMode
			} else if sameSite == "none" {
				ss = http.SameSiteNoneMode
			}
			c.SetSameSite(ss)
			c.SetCookie(CookieName, newID, CookieMaxAge, "/", domain, secure, true)
			c.Set("user_id", newID)
			c.Next()
			return
		}

		c.Set("user_id", cookie)
		c.Next()
	}
}
