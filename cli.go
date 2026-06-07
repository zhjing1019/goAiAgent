// 命令行模式（原来的 Todo CLI）
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func runCLI(app *TodoApp) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("=================================")
	fmt.Println("   📋 Go 待办清单（Todo List）")
	fmt.Println("=================================")
	printMenu()

	for {
		fmt.Print("\n请选择 (1-5): ")
		if !scanner.Scan() {
			break
		}
		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			handleAdd(scanner, app)
		case "2":
			fmt.Println("\n--- 待办列表 ---")
			app.List()
			fmt.Printf("未完成: %d 条\n", app.PendingCount())
		case "3":
			handleComplete(scanner, app)
		case "4":
			handleDelete(scanner, app)
		case "5":
			if err := app.Save(); err != nil {
				fmt.Println("保存失败:", err)
				return
			}
			fmt.Println("已保存，再见！")
			return
		default:
			fmt.Println("无效选项，请输入 1-5")
		}
	}
}

func printMenu() {
	fmt.Println("\n1. 添加待办")
	fmt.Println("2. 查看列表")
	fmt.Println("3. 标记完成")
	fmt.Println("4. 删除待办")
	fmt.Println("5. 保存并退出")
}

func handleAdd(scanner *bufio.Scanner, app *TodoApp) {
	fmt.Print("请输入待办内容: ")
	if !scanner.Scan() {
		return
	}
	title := strings.TrimSpace(scanner.Text())
	if title == "" {
		fmt.Println("内容不能为空")
		return
	}
	app.Add(title)
	fmt.Println("✓ 已添加")
}

func handleComplete(scanner *bufio.Scanner, app *TodoApp) {
	fmt.Print("请输入要完成的编号: ")
	id, ok := readID(scanner)
	if !ok {
		return
	}
	if err := app.Complete(id); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("✓ 已标记完成")
}

func handleDelete(scanner *bufio.Scanner, app *TodoApp) {
	fmt.Print("请输入要删除的编号: ")
	id, ok := readID(scanner)
	if !ok {
		return
	}
	if err := app.Delete(id); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("✓ 已删除")
}

func readID(scanner *bufio.Scanner) (int, bool) {
	if !scanner.Scan() {
		return 0, false
	}
	id, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil {
		fmt.Println("请输入有效数字")
		return 0, false
	}
	return id, true
}
