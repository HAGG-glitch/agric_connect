package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv  string
	AppPort string
	AppURL  string

	DatabaseURL string

	GroqAPIKey             string
	GroqBaseURL            string
	GroqChatModel          string
	GroqRequestTimeoutSecs int

	OpenMeteoBaseURL   string
	WeatherCacheMinutes int

	CookieSecure   bool
	CookieDomain   string
	CookieSameSite string

	MaxMessageLength           int
	MaxContextMessages         int
	MaxKnowledgeContextChars   int
	MaxDiagnosisContextChars   int
	GroqChatMaxOutputTokens    int
	GroqVisionMaxOutputTokens  int
	RateLimitPerMinute         int
	RateLimitAPIPerMinute      int
	RateLimitWeatherPerMinute  int

	GroqVisionModel          string
	GroqTranscriptionModel   string
	StorageDriver            string
	LocalUploadDir           string
	SupabaseURL              string
	SupabaseSecretKey        string
	SupabaseServiceRoleKey   string
	SupabaseStorageBucket    string
	MaxImageSizeMB           int
	MinImageWidth            int
	MinImageHeight           int
	MaxImagePixels           int64
	MaxAudioSizeMB           int
	MaxRecordingSeconds      int
	AllowedImageTypes        []string
	AllowedAudioTypes        []string
	DiagnosisRequestTimeout  int
	TranscriptionRequestTimeout int

	JWTAccessSecret   string
	JWTRefreshSecret  string
	JWTAccessDuration string
	JWTRefreshDuration string

	TranscriptionProvider string
	KrioSTTProvider       string
	HuggingFaceAPIKey     string
	HuggingFaceSTTModel   string
	HuggingFaceSTTTimeout int
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		AppPort:                getEnv("APP_PORT", "8081"),
		AppURL:                 getEnv("APP_URL", "http://localhost:8081"),
		DatabaseURL:            getEnv("DATABASE_URL", ""),
		GroqAPIKey:             getEnv("GROQ_API_KEY", ""),
		GroqBaseURL:            getEnv("GROQ_BASE_URL", "https://api.groq.com/openai/v1"),
		GroqChatModel:          getEnv("GROQ_CHAT_MODEL", "llama-3.1-8b-instant"),
		OpenMeteoBaseURL:       getEnv("OPEN_METEO_BASE_URL", "https://api.open-meteo.com/v1"),
		CookieDomain:           getEnv("COOKIE_DOMAIN", ""),
		CookieSameSite:         getEnv("COOKIE_SAME_SITE", "lax"),
		GroqRequestTimeoutSecs: getEnvInt("GROQ_REQUEST_TIMEOUT_SECONDS", 60),
		WeatherCacheMinutes:    getEnvInt("WEATHER_CACHE_MINUTES", 20),
		MaxMessageLength:           getEnvInt("MAX_MESSAGE_LENGTH", 4000),
		MaxContextMessages:         getEnvInt("MAX_CONTEXT_MESSAGES", 12),
		MaxKnowledgeContextChars:   getEnvInt("MAX_KNOWLEDGE_CONTEXT_CHARS", 2000),
		MaxDiagnosisContextChars:   getEnvInt("MAX_DIAGNOSIS_CONTEXT_CHARS", 500),
		GroqChatMaxOutputTokens:    getEnvInt("GROQ_CHAT_MAX_OUTPUT_TOKENS", 1024),
		GroqVisionMaxOutputTokens:  getEnvInt("GROQ_VISION_MAX_OUTPUT_TOKENS", 300),
		RateLimitPerMinute:         getEnvInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 20),
		RateLimitAPIPerMinute:      getEnvInt("RATE_LIMIT_API_PER_MINUTE", 20),
		RateLimitWeatherPerMinute:  getEnvInt("RATE_LIMIT_WEATHER_PER_MINUTE", 30),
		CookieSecure:                getEnvBool("COOKIE_SECURE", false),
		GroqVisionModel:             getEnv("GROQ_VISION_MODEL", "llama-3.2-11b-vision-preview"),
		GroqTranscriptionModel:      getEnv("GROQ_TRANSCRIPTION_MODEL", "whisper-large-v3"),
		StorageDriver:               getEnv("STORAGE_DRIVER", "local"),
		LocalUploadDir:              getEnv("LOCAL_UPLOAD_DIR", "./data/uploads"),
		SupabaseURL:                 getEnv("SUPABASE_URL", ""),
		SupabaseSecretKey:           getEnv("SUPABASE_SECRET_KEY", ""),
		SupabaseServiceRoleKey:      getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
		SupabaseStorageBucket:       getEnv("SUPABASE_STORAGE_BUCKET", "crop-diagnosis-images"),
		MaxImageSizeMB:              getEnvInt("MAX_IMAGE_SIZE_MB", 5),
		MinImageWidth:               getEnvInt("MIN_IMAGE_WIDTH", 256),
		MinImageHeight:              getEnvInt("MIN_IMAGE_HEIGHT", 256),
		MaxImagePixels:              int64(getEnvInt("MAX_IMAGE_PIXELS", 25000000)),
		MaxAudioSizeMB:              getEnvInt("MAX_AUDIO_SIZE_MB", 10),
		MaxRecordingSeconds:         getEnvInt("MAX_RECORDING_SECONDS", 60),
		AllowedImageTypes:           getEnvList("ALLOWED_IMAGE_TYPES", "image/jpeg,image/png,image/webp"),
		AllowedAudioTypes:           getEnvList("ALLOWED_AUDIO_TYPES", "audio/webm,audio/wav,audio/mpeg,audio/mp4,audio/ogg"),
		DiagnosisRequestTimeout:     getEnvInt("DIAGNOSIS_REQUEST_TIMEOUT_SECONDS", 90),
		TranscriptionRequestTimeout: getEnvInt("TRANSCRIPTION_REQUEST_TIMEOUT_SECONDS", 90),
		JWTAccessSecret:             getEnv("JWT_ACCESS_SECRET", ""),
		JWTRefreshSecret:            getEnv("JWT_REFRESH_SECRET", ""),
		JWTAccessDuration:           getEnv("JWT_ACCESS_DURATION", "15m"),
		JWTRefreshDuration:          getEnv("JWT_REFRESH_DURATION", "168h"),

		TranscriptionProvider: getEnv("TRANSCRIPTION_PROVIDER", "groq"),
		KrioSTTProvider:       getEnv("KRIO_STT_PROVIDER", "groq"),
		HuggingFaceAPIKey:     getEnv("HUGGINGFACE_API_KEY", ""),
		HuggingFaceSTTModel:   getEnv("HUGGINGFACE_STT_MODEL", "openai/whisper-large-v3"),
		HuggingFaceSTTTimeout: getEnvInt("HUGGINGFACE_STT_TIMEOUT_SECONDS", 60),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.AppEnv == "production" && cfg.GroqAPIKey == "" {
		return nil, fmt.Errorf("GROQ_API_KEY is required in production")
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

func (c *Config) AIAvailable() bool {
	return c.GroqAPIKey != ""
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvList(key string, fallback string) []string {
	v := os.Getenv(key)
	if v == "" {
		v = fallback
	}
	parts := strings.Split(v, ",")
	result := make([]string, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(p)
	}
	return result
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
