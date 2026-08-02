package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var robotCodePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type authResponse struct {
	Code int `json:"code"`
}

type proxyServer struct {
	authURL    string
	authClient *http.Client
}

func main() {
	listenAddress := envOrDefault("LISTEN_ADDRESS", ":9000")
	server := &proxyServer{
		authURL:    envOrDefault("ADMIN_AUTH_URL", "http://wechat-robot-admin-backend:9000/api/v1/user/self"),
		authClient: &http.Client{Timeout: 8 * time.Second},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("/api/v1/openchat/", server)

	log.Printf("openchat admin proxy listening on %s", listenAddress)
	if err := http.ListenAndServe(listenAddress, mux); err != nil {
		log.Fatal(err)
	}
}

func (s *proxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeError(w, http.StatusUnauthorized, 401, "登录信息已失效")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/openchat/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || !robotCodePattern.MatchString(parts[0]) {
		writeError(w, http.StatusBadRequest, 400, "机器人标识无效")
		return
	}

	target, err := url.Parse("http://client_" + parts[0] + ":9000")
	if err != nil {
		writeError(w, http.StatusInternalServerError, 500, "代理目标无效")
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = "/api/v1/" + parts[1]
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		log.Printf("proxy request failed: %v", err)
		writeError(w, http.StatusBadGateway, 502, "机器人客户端当前不可用")
	}
	proxy.ServeHTTP(w, r)
}

func (s *proxyServer) authorized(r *http.Request) bool {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, s.authURL, nil)
	if err != nil {
		return false
	}
	for _, header := range []string{"Cookie", "Authorization", "X-Api-Token"} {
		if value := r.Header.Get(header); value != "" {
			req.Header.Set(header, value)
		}
	}
	resp, err := s.authClient.Do(req)
	if err != nil {
		log.Printf("admin auth check failed: %v", err)
		return false
	}
	defer resp.Body.Close()
	var payload authResponse
	return json.NewDecoder(resp.Body).Decode(&payload) == nil && payload.Code == 200
}

func writeError(w http.ResponseWriter, status, code int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "message": message, "data": nil})
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
