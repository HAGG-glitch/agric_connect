package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNormalizePhone_AlreadyNormalized(t *testing.T) {
	s := &service{}
	input := "+23276100001"
	got := s.NormalizePhone(input)
	if got != input {
		t.Errorf("NormalizePhone(%q) = %q; want %q", input, got, input)
	}
}

func TestNormalizePhone_LocalFormat(t *testing.T) {
	s := &service{}
	input := "076100001"
	got := s.NormalizePhone(input)
	want := "+23276100001"
	if got != want {
		t.Errorf("NormalizePhone(%q) = %q; want %q", input, got, want)
	}
}

func TestNormalizePhone_CountryCodeWithoutPlus(t *testing.T) {
	s := &service{}
	input := "23276100001"
	got := s.NormalizePhone(input)
	want := "+23276100001"
	if got != want {
		t.Errorf("NormalizePhone(%q) = %q; want %q", input, got, want)
	}
}

func TestNormalizePhone_WithSpacesDashes(t *testing.T) {
	s := &service{}
	got := s.NormalizePhone(" 076-100-001 ")
	want := "+23276100001"
	if got != want {
		t.Errorf("NormalizePhone = %q; want %q", got, want)
	}
}

func TestNormalizePhone_InternationalWithDoubleZero(t *testing.T) {
	s := &service{}
	got := s.NormalizePhone("0023276100001")
	want := "+23276100001"
	if got != want {
		t.Errorf("NormalizePhone = %q; want %q", got, want)
	}
}

func TestNormalizePhone_WithParentheses(t *testing.T) {
	s := &service{}
	got := s.NormalizePhone("+232(76)100-001")
	want := "+23276100001"
	if got != want {
		t.Errorf("NormalizePhone = %q; want %q", got, want)
	}
}

func TestNormalizePhone_EmptyInput(t *testing.T) {
	s := &service{}
	got := s.NormalizePhone("")
	if got != "" {
		t.Errorf("NormalizePhone('') = %q; want ''", got)
	}
}

func TestNormalizePhone_BareNumber(t *testing.T) {
	s := &service{}
	got := s.NormalizePhone("76100001")
	want := "+23276100001"
	if got != want {
		t.Errorf("NormalizePhone(%q) = %q; want %q", "76100001", got, want)
	}
}

func TestGenerateRandomToken_NotEmpty(t *testing.T) {
	token, err := GenerateRandomToken()
	if err != nil {
		t.Fatalf("GenerateRandomToken() unexpected error: %v", err)
	}
	if len(token) == 0 {
		t.Fatal("GenerateRandomToken() returned empty token")
	}
}

func TestGenerateRandomToken_Length(t *testing.T) {
	token, err := GenerateRandomToken()
	if err != nil {
		t.Fatalf("GenerateRandomToken() unexpected error: %v", err)
	}
	if len(token) != 64 {
		t.Errorf("GenerateRandomToken() length = %d; want 64", len(token))
	}
}

func TestGenerateRandomToken_Uniqueness(t *testing.T) {
	t1, _ := GenerateRandomToken()
	t2, _ := GenerateRandomToken()
	if t1 == t2 {
		t.Fatal("GenerateRandomToken() returned duplicate tokens")
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	_, err := ValidateToken("invalid.jwt.token", "secret")
	if err == nil {
		t.Fatal("ValidateToken expected error for malformed token")
	}
}

func TestValidateToken_ValidToken(t *testing.T) {
	secret := "test-secret-key"
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "test-user-id",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	parsed, err := ValidateToken(tokenStr, secret)
	if err != nil {
		t.Fatalf("ValidateToken unexpected error: %v", err)
	}
	if parsed.Subject != "test-user-id" {
		t.Errorf("ValidateToken subject = %q; want %q", parsed.Subject, "test-user-id")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	secret := "test-secret-key"
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Subject:   "test-user-id",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	_, err = ValidateToken(tokenStr, secret)
	if err == nil {
		t.Fatal("ValidateToken expected error for expired token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	secret := "correct-secret"
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			Subject:   "test-user-id",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	_, err = ValidateToken(tokenStr, "wrong-secret")
	if err == nil {
		t.Fatal("ValidateToken expected error for wrong secret")
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	input := "test-token-value"
	h1 := hashToken(input)
	h2 := hashToken(input)
	if h1 != h2 {
		t.Fatal("hashToken should be deterministic but got different results")
	}
}

func TestHashToken_DifferentInputs(t *testing.T) {
	h1 := hashToken("token-a")
	h2 := hashToken("token-b")
	if h1 == h2 {
		t.Fatal("hashToken should produce different hashes for different inputs")
	}
}
