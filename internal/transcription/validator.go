package transcription

import (
	"fmt"
	"strings"
)

func ValidateLanguageHint(hint string) error {
	if hint == "" {
		return nil
	}
	if !ValidLanguageHints[strings.ToLower(hint)] {
		return fmt.Errorf("unsupported language hint: %s (must be english, krio, or auto)", hint)
	}
	return nil
}

func ValidateAudioContentType(contentType string, allowed []string) error {
	for _, a := range allowed {
		if strings.EqualFold(a, contentType) {
			return nil
		}
	}
	return fmt.Errorf("unsupported audio type: %s", contentType)
}
