package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Todo 一条待办事项：编号、标题、是否完成、创建时间
type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// TodoApp 管理所有待办，整个应用：Items 切片存所有待办，NextID 发新编号，file 存文件名

type TodoApp struct {
	Items  []Todo `json:"items"`
	NextID int    `json:"next_id"`
	file   string
}

// NewTodoApp 创建新应用，Items 空切片，NextID 1，file 传入
// 返回*TodoApp（指针），指向 &TodoApp{Items: []Todo{}, NextID: 1, file: file}
func NewTodoApp(file string) *TodoApp {
	return &TodoApp{
		// Items: []Todo{}：空切片，还没有待办
		Items:  []Todo{},
		NextID: 1,
		file:   file,
	}
}

// (a *TodoApp) 是指针接收者（第八课）：改 a.Items、a.NextID 会作用在原对象上。
func (a *TodoApp) Add(title string) {
	a.Items = append(a.Items, Todo{
		ID:        a.NextID,
		Title:     title,
		Done:      false,
		CreatedAt: time.Now(),
	})
	a.NextID++
}

func (a *TodoApp) List() {
	if len(a.Items) == 0 {
		fmt.Println("  （暂无待办，输入 1 添加一条）")
		return
	}
	for _, item := range a.Items {
		status := " "
		// if item.Done：完成显示 ✓，否则空格
		if item.Done {
			status = "✓"
		}
		fmt.Printf("  [%s] %d. %s  (%s)\n",
			// CreatedAt.Format(...)：把时间格式化成 06-06 16:02
			status, item.ID, item.Title, item.CreatedAt.Format("01-02 15:04"))
	}
}

func (a *TodoApp) Complete(id int) error {
	for i := range a.Items {
		if a.Items[i].ID == id {
			a.Items[i].Done = true
			return nil
		}
	}
	return fmt.Errorf("找不到编号 %d 的待办", id)
}

func (a *TodoApp) Delete(id int) error {
	for i, item := range a.Items {
		if item.ID == id {
			a.Items = append(a.Items[:i], a.Items[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("找不到编号 %d 的待办", id)
}

func (a *TodoApp) PendingCount() int {
	count := 0
	for _, item := range a.Items {
		if !item.Done {
			count++
		}
	}
	return count
}

// All 返回所有待办（给 Web API 用）
func (a *TodoApp) All() []Todo {
	return a.Items
}

func (a *TodoApp) Load() error {
	data, err := os.ReadFile(a.file)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, a)
}

func (a *TodoApp) Save() error {
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.file, data, 0644)
}
