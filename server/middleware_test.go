package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	rl := newRateLimiter()
	if rl == nil {
		t.Fatal("newRateLimiter returned nil")
	}
	if rl.attempts == nil {
		t.Error("attempts map is nil")
	}
	if rl.globalFailures == nil {
		t.Error("globalFailures slice is nil")
	}
}

func TestRateLimiterRecordFailure(t *testing.T) {
	rl := &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}

	ip := "192.168.1.1"

	// Record first failure
	rl.recordFailure(ip)

	if info, exists := rl.attempts[ip]; !exists {
		t.Error("IP should be in attempts map")
	} else if info.failCount != 1 {
		t.Errorf("failCount = %d, want 1", info.failCount)
	}

	// Record more failures
	for i := 0; i < 4; i++ {
		rl.recordFailure(ip)
	}

	info := rl.attempts[ip]
	if info.failCount != 5 {
		t.Errorf("failCount = %d, want 5", info.failCount)
	}

	// After 5 failures, should be locked
	locked, _, _ := rl.checkLocked(ip)
	if !locked {
		t.Error("IP should be locked after 5 failures")
	}
}

func TestRateLimiterRecordSuccess(t *testing.T) {
	rl := &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}

	ip := "192.168.1.1"

	// Record some failures
	for i := 0; i < 3; i++ {
		rl.recordFailure(ip)
	}

	// Verify failures recorded
	if rl.attempts[ip].failCount != 3 {
		t.Errorf("failCount = %d, want 3", rl.attempts[ip].failCount)
	}

	// Record success
	rl.recordSuccess(ip)

	// Fail count should be reset
	if rl.attempts[ip].failCount != 0 {
		t.Errorf("failCount after success = %d, want 0", rl.attempts[ip].failCount)
	}
}

func TestRateLimiterCheckLocked(t *testing.T) {
	rl := &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}

	ip := "192.168.1.1"

	// Not locked initially
	locked, _, _ := rl.checkLocked(ip)
	if locked {
		t.Error("IP should not be locked initially")
	}

	// Lock the IP
	rl.attempts[ip] = &attemptInfo{
		failCount:   10,
		lockedUntil: time.Now().Add(time.Hour),
	}

	locked, remaining, lockType := rl.checkLocked(ip)
	if !locked {
		t.Error("IP should be locked")
	}
	if lockType != "ip" {
		t.Errorf("lockType = %s, want 'ip'", lockType)
	}
	if remaining <= 0 {
		t.Error("remaining time should be positive")
	}
}

func TestRateLimiterGlobalLockout(t *testing.T) {
	rl := &rateLimiter{
		attempts:          make(map[string]*attemptInfo),
		globalFailures:    make([]time.Time, 0),
		globalLockedUntil: time.Now().Add(time.Hour),
	}

	ip := "192.168.1.1"
	locked, _, lockType := rl.checkLocked(ip)

	if !locked {
		t.Error("Should be globally locked")
	}
	if lockType != "global" {
		t.Errorf("lockType = %s, want 'global'", lockType)
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	rl := &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}

	// Add old entry
	oldIP := "10.0.0.1"
	rl.attempts[oldIP] = &attemptInfo{
		failCount:   0,
		lockedUntil: time.Now().Add(-time.Hour),
	}

	// Add recent entry
	recentIP := "10.0.0.2"
	rl.attempts[recentIP] = &attemptInfo{
		failCount:   5,
		lockedUntil: time.Now().Add(time.Hour),
	}

	// Add old global failures
	rl.globalFailures = []time.Time{
		time.Now().Add(-10 * time.Minute),
		time.Now().Add(-1 * time.Minute),
	}

	rl.cleanup()

	// Old entry should be removed
	if _, exists := rl.attempts[oldIP]; exists {
		t.Error("Old IP entry should have been cleaned up")
	}

	// Recent entry should remain
	if _, exists := rl.attempts[recentIP]; !exists {
		t.Error("Recent IP entry should not have been cleaned up")
	}

	// Recent global failure should remain
	if len(rl.globalFailures) != 1 {
		t.Errorf("globalFailures length = %d, want 1", len(rl.globalFailures))
	}
}

func TestRateLimiterPruneGlobalFailures(t *testing.T) {
	rl := &rateLimiter{
		attempts: make(map[string]*attemptInfo),
		globalFailures: []time.Time{
			time.Now().Add(-10 * time.Minute), // Outside window
			time.Now().Add(-3 * time.Minute),  // Inside window
			time.Now().Add(-1 * time.Minute),  // Inside window
		},
	}

	rl.pruneGlobalFailures(time.Now())

	if len(rl.globalFailures) != 2 {
		t.Errorf("globalFailures length = %d, want 2", len(rl.globalFailures))
	}
}

// Helper function to create a test server for middleware tests
func createTestServer() *Server {
	return &Server{
		options: &Options{},
	}
}

func TestWrapHeaders(t *testing.T) {
	server := createTestServer()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := server.wrapHeaders(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Header().Get("Server") != "WebTmux" {
		t.Errorf("Server header = %s, want 'WebTmux'", rr.Header().Get("Server"))
	}
}

func TestWrapLogger(t *testing.T) {
	server := createTestServer()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := server.wrapLogger(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Should not panic
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestWrapBasicAuthValid(t *testing.T) {
	// Use a fresh rate limiter for testing
	oldLimiter := authRateLimiter
	authRateLimiter = &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}
	defer func() { authRateLimiter = oldLimiter }()

	server := createTestServer()
	credential := "admin:password"

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := server.wrapBasicAuth(handler, credential)

	// Create valid auth header
	auth := base64.StdEncoding.EncodeToString([]byte(credential))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic "+auth)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("Handler should have been called with valid credentials")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestWrapBasicAuthInvalid(t *testing.T) {
	// Use a fresh rate limiter for testing
	oldLimiter := authRateLimiter
	authRateLimiter = &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}
	defer func() { authRateLimiter = oldLimiter }()

	server := createTestServer()
	credential := "admin:password"

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	wrapped := server.wrapBasicAuth(handler, credential)

	// Create invalid auth header
	auth := base64.StdEncoding.EncodeToString([]byte("wrong:creds"))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic "+auth)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if handlerCalled {
		t.Error("Handler should not have been called with invalid credentials")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestWrapBasicAuthMissingHeader(t *testing.T) {
	// Use a fresh rate limiter for testing
	oldLimiter := authRateLimiter
	authRateLimiter = &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}
	defer func() { authRateLimiter = oldLimiter }()

	server := createTestServer()
	credential := "admin:password"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	wrapped := server.wrapBasicAuth(handler, credential)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if rr.Header().Get("WWW-Authenticate") == "" {
		t.Error("WWW-Authenticate header should be set")
	}
}

func TestWrapBasicAuthLockout(t *testing.T) {
	// Use a fresh rate limiter for testing
	oldLimiter := authRateLimiter
	authRateLimiter = &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}
	defer func() { authRateLimiter = oldLimiter }()

	server := createTestServer()
	credential := "admin:password"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	wrapped := server.wrapBasicAuth(handler, credential)

	// Lock out the IP
	authRateLimiter.attempts["192.0.2.1"] = &attemptInfo{
		failCount:   10,
		lockedUntil: time.Now().Add(time.Hour),
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusTooManyRequests)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header should be set")
	}
}

func TestWrapBasicAuthXForwardedFor(t *testing.T) {
	// Use a fresh rate limiter for testing
	oldLimiter := authRateLimiter
	authRateLimiter = &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}
	defer func() { authRateLimiter = oldLimiter }()

	server := createTestServer()
	credential := "admin:password"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	wrapped := server.wrapBasicAuth(handler, credential)

	// Lock out the forwarded IP
	authRateLimiter.attempts["10.0.0.1"] = &attemptInfo{
		failCount:   10,
		lockedUntil: time.Now().Add(time.Hour),
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Status code = %d, want %d (should use X-Forwarded-For IP)", rr.Code, http.StatusTooManyRequests)
	}
}

func TestWrapBasicAuthInvalidBase64(t *testing.T) {
	// Use a fresh rate limiter for testing
	oldLimiter := authRateLimiter
	authRateLimiter = &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}
	defer func() { authRateLimiter = oldLimiter }()

	server := createTestServer()
	credential := "admin:password"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	wrapped := server.wrapBasicAuth(handler, credential)

	// Send invalid base64
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic not-valid-base64!!!")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %d, want %d for invalid base64", rr.Code, http.StatusInternalServerError)
	}
}

func TestWrapBasicAuthGlobalLockout(t *testing.T) {
	// Use a fresh rate limiter for testing
	oldLimiter := authRateLimiter
	authRateLimiter = &rateLimiter{
		attempts:          make(map[string]*attemptInfo),
		globalFailures:    make([]time.Time, 0),
		globalLockedUntil: time.Now().Add(time.Hour),
	}
	defer func() { authRateLimiter = oldLimiter }()

	server := createTestServer()
	credential := "admin:password"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	wrapped := server.wrapBasicAuth(handler, credential)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Status code = %d, want %d for global lockout", rr.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimiterRecordFailureTriggersLockout(t *testing.T) {
	rl := &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}

	ip := "192.168.1.1"

	// Record 5 failures to trigger first lockout
	for i := 0; i < 5; i++ {
		rl.recordFailure(ip)
	}

	// Should be locked
	locked, _, _ := rl.checkLocked(ip)
	if !locked {
		t.Error("IP should be locked after 5 failures")
	}

	// Record more failures to trigger longer lockout
	for i := 0; i < 5; i++ {
		rl.recordFailure(ip)
	}

	// Should still be locked with longer duration
	locked, duration, _ := rl.checkLocked(ip)
	if !locked {
		t.Error("IP should be locked after 10 failures")
	}
	if duration < time.Minute {
		t.Errorf("Duration = %v, should be at least 1 minute for 10 failures", duration)
	}
}

// Benchmark rate limiter operations
func BenchmarkRateLimiterCheckLocked(b *testing.B) {
	rl := &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.checkLocked("192.168.1.1")
	}
}

func BenchmarkRateLimiterRecordFailure(b *testing.B) {
	rl := &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.recordFailure("192.168.1.1")
	}
}
