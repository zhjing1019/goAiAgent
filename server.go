package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// runServer 启动 Web API（第十一课：net/http）
func runServer(app *TodoApp, addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /todos/stats", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"total":   len(app.All()),
			"pending": app.PendingCount(),
		})
	})

	mux.HandleFunc("GET /todos", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"items":   app.All(),
			"pending": app.PendingCount(),
		})
	})

	mux.HandleFunc("POST /todos", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "JSON 格式错误")
			return
		}
		if body.Title == "" {
			writeError(w, http.StatusBadRequest, "title 不能为空")
			return
		}
		app.Add(body.Title)
		if err := app.Save(); err != nil {
			writeError(w, http.StatusInternalServerError, "保存失败")
			return
		}
		item := app.All()[len(app.All())-1]
		writeJSON(w, http.StatusCreated, item)
	})

	mux.HandleFunc("PATCH /todos/{id}/complete", func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := app.Complete(id); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if err := app.Save(); err != nil {
			writeError(w, http.StatusInternalServerError, "保存失败")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "已标记完成"})
	})

	mux.HandleFunc("DELETE /todos/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := app.Delete(id); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if err := app.Save(); err != nil {
			writeError(w, http.StatusInternalServerError, "保存失败")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "已删除"})
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Todo API 运行中。试试 GET /todos")
	})

	fmt.Println("🚀 Web 服务已启动:", addr)
	fmt.Println("   GET    /todos/stats")
	fmt.Println("   GET    /todos")
	fmt.Println("   POST   /todos          body: {\"title\":\"学习 HTTP\"}")
	fmt.Println("   PATCH  /todos/1/complete")
	fmt.Println("   DELETE /todos/1")
	return http.ListenAndServe(addr, mux)
}

func parseID(s string) (int, error) {
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("无效的 id: %s", s)
	}
	return id, nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
