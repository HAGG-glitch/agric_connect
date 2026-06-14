package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/agriconnect-ai/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	authSvc  auth.Service
	secure   bool
	domain   string
	sameSite string
}

func NewAuthHandler(authSvc auth.Service, secure bool, domain string, sameSite string) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, secure: secure, domain: domain, sameSite: sameSite}
}

func (h *AuthHandler) RegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{
		"Title": "AgriConnect AI - Register",
		"Year":  time.Now().Year(),
	})
}

func (h *AuthHandler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"Title": "AgriConnect AI - Login",
		"Year":  time.Now().Year(),
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		FullName          string `json:"full_name" form:"full_name"`
		PhoneNumber       string `json:"phone_number" form:"phone_number"`
		District          string `json:"district" form:"district"`
		PreferredLanguage string `json:"preferred_language" form:"preferred_language"`
		Password          string `json:"password" form:"password"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	tokens, err := h.authSvc.Register(c.Request.Context(), auth.RegisterInput{
		FullName:          req.FullName,
		PhoneNumber:       req.PhoneNumber,
		District:          req.District,
		PreferredLanguage: req.PreferredLanguage,
		Password:          req.Password,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	anonymousIDStr, _ := c.Get("user_id")
	anonymousID, _ := uuid.Parse(anonymousIDStr.(string))

	h.setCookies(c, tokens)
	h.tryTransfer(c, anonymousID, tokens.User.ID)

	c.JSON(http.StatusCreated, tokens)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		PhoneNumber string `json:"phone_number" form:"phone_number"`
		Password    string `json:"password" form:"password"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	tokens, err := h.authSvc.Login(c.Request.Context(), auth.LoginInput{
		PhoneNumber: req.PhoneNumber,
		Password:    req.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.setCookies(c, tokens)
	c.JSON(http.StatusOK, tokens)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshTokenStr, err := c.Cookie("refresh_token")
	if err != nil || refreshTokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No refresh token"})
		return
	}

	tokens, err := h.authSvc.RefreshToken(c.Request.Context(), refreshTokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.setCookies(c, tokens)
	c.JSON(http.StatusOK, tokens)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))
	refreshTokenStr, err := c.Cookie("refresh_token")
	if err == nil && refreshTokenStr != "" {
		claims, parseErr := auth.ValidateToken(refreshTokenStr, "")
		if parseErr == nil && claims != nil {
			if rtID, parseErr2 := uuid.Parse(claims.ID); parseErr2 == nil {
				_ = h.authSvc.Logout(c.Request.Context(), userID, rtID)
			}
		}
	}

	h.clearCookies(c)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	user, err := h.authSvc.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) setCookies(c *gin.Context, tokens *auth.TokenPair) {
	ss := h.parseSameSite()

	c.SetSameSite(ss)
	c.SetCookie("access_token", tokens.AccessToken, 900, "/", h.domain, h.secure, true)

	c.SetSameSite(ss)
	c.SetCookie("refresh_token", tokens.RefreshToken, 604800, "/api/v1/auth", h.domain, h.secure, true)

	c.SetSameSite(ss)
	c.SetCookie("refresh_token", tokens.RefreshToken, 604800, "/login", h.domain, h.secure, true)
}

func (h *AuthHandler) clearCookies(c *gin.Context) {
	ss := h.parseSameSite()
	c.SetSameSite(ss)
	c.SetCookie("access_token", "", -1, "/", h.domain, h.secure, true)
	c.SetSameSite(ss)
	c.SetCookie("refresh_token", "", -1, "/", h.domain, h.secure, true)
}

func (h *AuthHandler) parseSameSite() http.SameSite {
	switch h.sameSite {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func (h *AuthHandler) tryTransfer(c *gin.Context, anonymousID, userID uuid.UUID) {
	if anonymousID == uuid.Nil || userID == uuid.Nil {
		return
	}
	if err := h.authSvc.TransferAnonymousData(c.Request.Context(), anonymousID, userID); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
	}
}


