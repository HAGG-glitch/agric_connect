package transcription

type TranscriptionInput struct {
	Audio        []byte
	AudioType    string
	LanguageHint string
	SizeBytes    int64
}

type TranscriptionResponse struct {
	Transcript         string `json:"transcript"`
	DetectedLanguage   string `json:"detected_language"`
	RequiresConfirmation bool `json:"requires_confirmation"`
	ExperimentalKrio   bool   `json:"experimental_krio"`
}

var ValidLanguageHints = map[string]bool{
	"english": true,
	"krio":    true,
	"auto":    true,
}
