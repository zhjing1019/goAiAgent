// Package mysql 实现 SessionStore 的 MySQL 版本（第 5 步）。
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zhjing1019/goAiAgent/internal/llm"
	"github.com/zhjing1019/goAiAgent/internal/store"

	_ "github.com/go-sql-driver/mysql" // 注册 MySQL 驱动
)

// Store MySQL 会话存储。
type Store struct {
	db *sql.DB
}

// Open 连接 MySQL 并返回 Store。
//
// dsn 示例：
//
//	user:pass@tcp(127.0.0.1:3306)/go_agent?parseTime=true&charset=utf8mb4&loc=Local
func Open(dsn string) (*Store, error) {
	// 打开 MySQL 连接
	db, err := sql.Open("mysql", dsn)
	// 如果打开连接失败，返回错误
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	// 设置连接池最大连接数
	db.SetMaxOpenConns(10)
	// 设置连接池最大空闲连接数
	db.SetMaxIdleConns(5)
	// 设置连接池最大生命周期
	db.SetConnMaxLifetime(time.Hour)
	// 创建一个超时上下文

	// 创建一个超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// 延迟取消上下文
	defer cancel()
	// ping MySQL 连接
	// 	向 MySQL 数据库发送一个 “心跳包”，看看数据库连没连上、活不活。
	// 如果连不上，就关闭连接，返回错误。
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return &Store{db: db}, nil
}

// Close 关闭连接池。
func (s *Store) Close() error {
	return s.db.Close()
}

// DB 暴露底层连接（测试用）。
func (s *Store) DB() *sql.DB {
	return s.db
}

// Migrate 自动建表。
func (s *Store) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			title VARCHAR(255) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS messages (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			session_id VARCHAR(36) NOT NULL,
			seq INT NOT NULL,
			role VARCHAR(20) NOT NULL,
			content MEDIUMTEXT,
			tool_calls JSON NULL,
			tool_call_id VARCHAR(64) NULL,
			tool_name VARCHAR(64) NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_messages_session (session_id, seq),
			CONSTRAINT fk_messages_session FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// CreateSession 新建会话。
func (s *Store) CreateSession(ctx context.Context, title string) (string, error) {
	id := uuid.NewString()
	title = truncateTitle(title, 50)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, title) VALUES (?, ?)`,
		id, title,
	)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return id, nil
}

// LoadMessages 加载会话全部消息。
func (s *Store) LoadMessages(ctx context.Context, sessionID string) ([]llm.Message, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT role, content, tool_calls, tool_call_id, tool_name
		FROM messages
		WHERE session_id = ?
		ORDER BY seq ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load messages: %w", err)
	}
	defer rows.Close()

	var out []llm.Message
	for rows.Next() {
		var role, content string
		var toolCalls []byte
		var toolCallID, toolName sql.NullString
		if err := rows.Scan(&role, &content, &toolCalls, &toolCallID, &toolName); err != nil {
			return nil, err
		}
		msg, err := decodeMessage(role, content, toolCalls, toolCallID, toolName)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, rows.Err()
}

// AppendMessage 追加一条消息。
func (s *Store) AppendMessage(ctx context.Context, sessionID string, msg llm.Message) error {
	content, toolCallsJSON, toolCallID, toolName, err := encodeMessage(msg)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var nextSeq int
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(seq), 0) + 1 FROM messages WHERE session_id = ?`,
		sessionID,
	).Scan(&nextSeq)
	if err != nil {
		return fmt.Errorf("next seq: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO messages (session_id, seq, role, content, tool_calls, tool_call_id, tool_name)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, sessionID, nextSeq, msg.Role, content, nullJSON(toolCallsJSON), nullString(toolCallID), nullString(toolName))
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}

	_, err = tx.ExecContext(ctx, `UPDATE sessions SET updated_at = NOW() WHERE id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}

	return tx.Commit()
}

// DeleteSession 删除会话。
func (s *Store) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
	return err
}

// ListSessions 列出最近会话。
func (s *Store) ListSessions(ctx context.Context, limit int) ([]store.Session, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, created_at, updated_at
		FROM sessions
		ORDER BY updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// 创建一个空数组，用于存储会话	
	var out []store.Session
	// 遍历查询结果
	for rows.Next() {
		// 创建一个会话对象
		var sess store.Session
		if err := rows.Scan(&sess.ID, &sess.Title, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

func nullJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
