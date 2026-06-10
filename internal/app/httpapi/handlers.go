package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// routes 注册路由
func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /", s.handleRoot)
	s.mux.HandleFunc("POST /api/chat", s.handleChat)
	s.mux.HandleFunc("GET /api/sessions", s.handleListSessions)
	s.mux.HandleFunc("POST /api/knowledge/search", s.handleKnowledgeSearch)
}

// handleRoot 处理根路径请求
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"service": "go-ai-agent",
		"docs":    "GET /api/health, POST /api/chat",
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	st := s.app.Status()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"env":           st.Env,
		"mysql":         st.MySQLEnabled,
		"rag":           st.RAGEnabled,
		"redis":         st.RedisEnabled,
		"session_cache": st.SessionCacheEnabled,
		"rate_limit":    st.RateLimitEnabled,
	})
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "JSON 格式错误")
		return
	}
	if body.Message == "" {
		writeError(w, http.StatusBadRequest, "message 不能为空")
		return
	}

	result, err := s.app.RunChat(r.Context(), body.SessionID, body.Message)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"reply":      result.Reply,
		"session_id": result.SessionID,
	})
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	if !s.app.Status().MySQLEnabled {
		writeError(w, http.StatusServiceUnavailable, "未启用 MySQL")
		return
	}
	limit := 20
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := s.app.ListSessions(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": list})
}

func (s *Server) handleKnowledgeSearch(w http.ResponseWriter, r *http.Request) {
	if !s.app.Status().RAGEnabled {
		writeError(w, http.StatusServiceUnavailable, "RAG 未启用")
		return
	}
	var body struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "JSON 格式错误")
		return
	}
	if body.Query == "" {
		writeError(w, http.StatusBadRequest, "query 不能为空")
		return
	}
	chunks, err := s.app.SearchKnowledge(r.Context(), body.Query, body.TopK)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": chunks})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
