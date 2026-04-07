package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type server struct {
	adminAPIBase string
	sessions     *sessionManager
	client       *http.Client
	csrfSecret   []byte
	sessionTTL   time.Duration
}

type sessionManager struct {
	secret []byte
}

func main() {
	adminAPI := getenv("ADMIN_API_BASE", "http://localhost:8080")
	port := getenv("WEBUI_PORT", "8090")
	secret := []byte(os.Getenv("WEBUI_SESSION_SECRET"))
	tlsCert := os.Getenv("WEBUI_TLS_CERT")
	tlsKey := os.Getenv("WEBUI_TLS_KEY")
	csrfSecret := []byte(os.Getenv("WEBUI_CSRF_SECRET"))
	sessionTTL := parseDurationOr("WEBUI_SESSION_TTL", time.Hour)

	if _, err := url.Parse(adminAPI); err != nil {
		log.Fatalf("invalid ADMIN_API_BASE: %v", err)
	}

	s := &server{
		adminAPIBase: strings.TrimRight(adminAPI, "/"),
		sessions:     newSessionManager(secret),
		client:       &http.Client{Timeout: 15 * time.Second},
		csrfSecret:   csrfSecret,
		sessionTTL:   sessionTTL,
	}

	http.HandleFunc("/auth/login", s.loginHandler)
	http.HandleFunc("/auth/logout", s.logoutHandler)
	http.HandleFunc("/auth/me", s.meHandler)
	http.HandleFunc("/api/", s.apiProxy)
	http.Handle("/", http.FileServer(http.Dir("web/ui")))

	log.Printf("Starting web UI server on :%s (admin API: %s)", port, adminAPI)
	var err error
	if tlsCert != "" && tlsKey != "" {
		err = http.ListenAndServeTLS(":"+port, tlsCert, tlsKey, nil)
	} else {
		err = http.ListenAndServe(":"+port, nil)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func newSessionManager(secret []byte) *sessionManager {
	if len(secret) == 0 {
		secret = mustRandom(32)
	}
	return &sessionManager{secret: secret}
}

func mustRandom(n int) []byte {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return buf
}

func (s *sessionManager) sign(token string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(token))
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(token)) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func (s *sessionManager) verify(cookieVal string) (string, bool) {
	parts := strings.Split(cookieVal, ".")
	if len(parts) != 2 {
		return "", false
	}
	tokenBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(tokenBytes)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return "", false
	}
	return string(tokenBytes), true
}

// loginHandler stores a signed session cookie with the provided bearer token.
func (s *server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Token) == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		return
	}
	signed := s.sessions.sign(strings.TrimSpace(body.Token))
	csrfRaw := base64.RawURLEncoding.EncodeToString(mustRandom(18))
	csrfSigned := s.signCSRF(csrfRaw)
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    signed,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		Expires:  time.Now().Add(s.sessionTTL),
	}
	http.SetCookie(w, cookie)
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfSigned,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		Expires:  time.Now().Add(s.sessionTTL),
	})
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("failed to encode login response: %v", err)
	}
}

// logoutHandler clears the session cookie.
func (s *server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	}
	http.SetCookie(w, cookie)
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Expires:  time.Unix(0, 0),
	})
	w.WriteHeader(http.StatusNoContent)
}

// meHandler validates the stored session by probing the admin health endpoint.
func (s *server) meHandler(w http.ResponseWriter, r *http.Request) {
	token, ok := s.tokenFromRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := s.checkHealth(r.Context(), token); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("failed to encode me response: %v", err)
	}
}

// apiProxy forwards /api/* calls to the admin API, enforcing presence of a valid session token.
func (s *server) apiProxy(w http.ResponseWriter, r *http.Request) {
	token, ok := s.tokenFromRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !s.checkCSRF(w, r) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	target := s.adminAPIBase + path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Header = r.Header.Clone()
	req.Header.Set("Authorization", "Bearer "+token)
	req.Host = ""

	resp, err := s.client.Do(req)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("failed to copy response body: %v", err)
	}
	log.Printf("proxy %s %s -> %d", r.Method, path, resp.StatusCode)
}

func (s *server) tokenFromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return "", false
	}
	token, ok := s.sessions.verify(cookie.Value)
	return token, ok
}

func (s *server) checkHealth(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.adminAPIBase+"/admin/health", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("unauthorized")
	}
	return nil
}

func copyHeaders(dst, src http.Header) {
	for k, vals := range src {
		if strings.EqualFold(k, "Content-Length") {
			continue
		}
		for _, v := range vals {
			dst.Add(k, v)
		}
	}
}

func parseDurationOr(env string, def time.Duration) time.Duration {
	if v := os.Getenv(env); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil && d > 0 {
			return d
		}
		log.Printf("invalid %s, using default %s", env, def)
	}
	return def
}

func (s *server) signCSRF(raw string) string {
	mac := hmac.New(sha256.New, s.csrfSecretValue())
	mac.Write([]byte(raw))
	return base64.RawURLEncoding.EncodeToString([]byte(raw)) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *server) verifyCSRF(signed string) (string, bool) {
	parts := strings.Split(signed, ".")
	if len(parts) != 2 {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	mac := hmac.New(sha256.New, s.csrfSecretValue())
	mac.Write(raw)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return "", false
	}
	return string(raw), true
}

func (s *server) csrfSecretValue() []byte {
	if len(s.csrfSecret) == 0 {
		s.csrfSecret = mustRandom(32)
	}
	return s.csrfSecret
}

func (s *server) checkCSRF(w http.ResponseWriter, r *http.Request) bool {
	// Safe methods skip CSRF
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	cookie, err := r.Cookie("csrf_token")
	if err != nil {
		http.Error(w, "csrf required", http.StatusForbidden)
		return false
	}
	header := r.Header.Get("X-CSRF-Token")
	if header == "" || header != cookie.Value {
		http.Error(w, "csrf mismatch", http.StatusForbidden)
		return false
	}
	if _, ok := s.verifyCSRF(cookie.Value); !ok {
		http.Error(w, "csrf invalid", http.StatusForbidden)
		return false
	}
	return true
}
