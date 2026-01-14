package server

import (
	"crypto/subtle"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// rateLimiter provides brute force protection for authentication
type rateLimiter struct {
	// Per-IP tracking
	attempts map[string]*attemptInfo

	// Global tracking (sliding window)
	globalFailures    []time.Time
	globalLockedUntil time.Time

	mu sync.RWMutex
}

type attemptInfo struct {
	failCount   int
	lockedUntil time.Time
	lastSeen    time.Time
}

// Per-IP lockout thresholds
var ipLockoutRules = []struct {
	attempts int
	duration time.Duration
}{
	{5, 1 * time.Minute},
	{10, 5 * time.Minute},
	{20, 15 * time.Minute},
}

// Global lockout thresholds (higher initial threshold)
var globalLockoutRules = []struct {
	failures int
	duration time.Duration
}{
	{100, 2 * time.Minute},
	{200, 10 * time.Minute},
	{500, 30 * time.Minute},
}

const globalWindowDuration = 5 * time.Minute

func newRateLimiter() *rateLimiter {
	rl := &rateLimiter{
		attempts:       make(map[string]*attemptInfo),
		globalFailures: make([]time.Time, 0),
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// cleanupLoop periodically removes old entries
func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes expired entries
func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-30 * time.Minute)

	// Clean up per-IP entries
	for ip, info := range rl.attempts {
		if info.lastSeen.Before(cutoff) && now.After(info.lockedUntil) {
			delete(rl.attempts, ip)
		}
	}

	// Clean up global failures outside window
	rl.pruneGlobalFailures(now)
}

// pruneGlobalFailures removes failures outside the sliding window
func (rl *rateLimiter) pruneGlobalFailures(now time.Time) {
	cutoff := now.Add(-globalWindowDuration)
	newFailures := make([]time.Time, 0, len(rl.globalFailures))
	for _, t := range rl.globalFailures {
		if t.After(cutoff) {
			newFailures = append(newFailures, t)
		}
	}
	rl.globalFailures = newFailures
}

// checkLocked returns lockout duration if IP or global is locked
func (rl *rateLimiter) checkLocked(ip string) (bool, time.Duration, string) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()

	// Check global lockout first
	if now.Before(rl.globalLockedUntil) {
		remaining := rl.globalLockedUntil.Sub(now)
		return true, remaining, "global"
	}

	// Check per-IP lockout
	if info, exists := rl.attempts[ip]; exists {
		if now.Before(info.lockedUntil) {
			remaining := info.lockedUntil.Sub(now)
			return true, remaining, "ip"
		}
	}

	return false, 0, ""
}

// recordFailure records a failed login attempt
func (rl *rateLimiter) recordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Record per-IP failure
	info, exists := rl.attempts[ip]
	if !exists {
		info = &attemptInfo{}
		rl.attempts[ip] = info
	}

	// Reset if lockout expired
	if now.After(info.lockedUntil) && info.failCount > 0 {
		// Keep some history but allow attempts
	}

	info.failCount++
	info.lastSeen = now

	// Apply per-IP lockout
	for _, rule := range ipLockoutRules {
		if info.failCount >= rule.attempts {
			info.lockedUntil = now.Add(rule.duration)
		}
	}

	// Record global failure
	rl.globalFailures = append(rl.globalFailures, now)
	rl.pruneGlobalFailures(now)

	// Check global lockout
	failureCount := len(rl.globalFailures)
	for _, rule := range globalLockoutRules {
		if failureCount >= rule.failures {
			rl.globalLockedUntil = now.Add(rule.duration)
		}
	}

	log.Printf("Auth failure from %s (IP attempts: %d, global failures: %d)", ip, info.failCount, failureCount)
}

// recordSuccess resets the per-IP counter on successful login
func (rl *rateLimiter) recordSuccess(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if info, exists := rl.attempts[ip]; exists {
		info.failCount = 0
		info.lockedUntil = time.Time{}
		info.lastSeen = time.Now()
	}
}

// Global rate limiter instance
var authRateLimiter = newRateLimiter()

func (server *Server) wrapLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &logResponseWriter{w, 200}
		handler.ServeHTTP(rw, r)
		log.Printf("%s %d %s %s", r.RemoteAddr, rw.status, r.Method, r.URL.Path)
	})
}

func (server *Server) wrapHeaders(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "WebTmux")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		handler.ServeHTTP(w, r)
	})
}

func (server *Server) wrapBasicAuth(handler http.Handler, credential string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !server.requireBasicAuth(w, r, credential) {
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func (server *Server) requireBasicAuth(w http.ResponseWriter, r *http.Request, credential string) bool {
	ip := server.requestIP(r)
	if locked, remaining, lockType := authRateLimiter.checkLocked(ip); locked {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(remaining.Seconds())+1))
		if lockType == "global" {
			log.Printf("Global lockout active, rejected %s (retry in %v)", ip, remaining)
			http.Error(w, "Too many failed login attempts. Service temporarily locked.", http.StatusTooManyRequests)
		} else {
			log.Printf("IP %s locked out (retry in %v)", ip, remaining)
			http.Error(w, "Too many failed login attempts. Try again later.", http.StatusTooManyRequests)
		}
		return false
	}

	username, password, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="WebTmux"`)
		http.Error(w, "Bad Request", http.StatusUnauthorized)
		return false
	}

	if !server.checkCredential(credential, username, password) {
		authRateLimiter.recordFailure(ip)
		w.Header().Set("WWW-Authenticate", `Basic realm="WebTmux"`)
		http.Error(w, "Authorization failed", http.StatusUnauthorized)
		return false
	}

	authRateLimiter.recordSuccess(ip)
	log.Printf("Basic Authentication Succeeded: %s", r.RemoteAddr)
	return true
}

func (server *Server) requestIP(r *http.Request) string {
	if server.options != nil && server.options.TrustProxy {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip := strings.TrimSpace(strings.Split(forwarded, ",")[0])
			if ip != "" {
				return ip
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (server *Server) checkCredential(credential, username, password string) bool {
	payload := []byte(username + ":" + password)
	credBytes := []byte(credential)
	if len(payload) != len(credBytes) {
		return false
	}
	return subtle.ConstantTimeCompare(payload, credBytes) == 1
}
