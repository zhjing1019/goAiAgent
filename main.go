// Todo 项目入口：默认 Web API，加 -cli 走命令行
package main

import (
	"flag"
	"fmt"
	"os"
)

const dataFile = "todos.json"

func main() {
	cliMode := flag.Bool("cli", false, "使用命令行模式")
	addr := flag.String("addr", ":8080", "Web 服务监听地址")
	flag.Parse()

	app := NewTodoApp(dataFile)
	if err := app.Load(); err != nil {
		fmt.Println("读取数据失败:", err)
		os.Exit(1)
	}

	if *cliMode {
		runCLI(app)
		return
	}

	if err := runServer(app, *addr); err != nil {
		fmt.Println("服务启动失败:", err)
		os.Exit(1)
	}
}
