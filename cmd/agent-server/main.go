// 第 8 步：Agent HTTP API 服务入口
//
// 运行：
//
//	make run-server-dev
//	curl http://localhost:8080/api/health
//	curl -X POST http://localhost:8080/api/chat -H 'Content-Type: application/json' -d '{"message":"现在几点？"}'
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhjing1019/goAiAgent/internal/app"
	"github.com/zhjing1019/goAiAgent/internal/app/httpapi"
	"github.com/zhjing1019/goAiAgent/internal/config"
)

func main() {
	ctx := context.Background()
	application, err := app.NewFromEnv(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()

	application.Status().PrintStartup()

	addr, err := config.LoadHTTPAddr()
	if err != nil {
		log.Fatal(err)
	}

	srv := httpapi.New(application)
	go func() {
		if err := srv.ListenAndServe(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}
