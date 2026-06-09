package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// EnsureDatabase 若 DSN 中的数据库不存在则自动创建。
//
// 程序 Migrate 只会建表，不会建库；首次运行前调用本函数可避免 Error 1049。
func EnsureDatabase(dsn string) error {
	dbName, serverDSN, err := splitDSN(dsn)
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", serverDSN)
	if err != nil {
		return fmt.Errorf("open mysql server: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping mysql server: %w", err)
	}

	query := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci",
		dbName,
	)
	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("create database %s: %w", dbName, err)
	}
	return nil
}
// 分割 DSN 字符串，返回数据库名称和服务器 DSN
func splitDSN(dsn string) (dbName, serverDSN string, err error) {
	slash := strings.LastIndex(dsn, "/")
	if slash < 0 {
		return "", "", fmt.Errorf("invalid MYSQL_DSN: missing database name")
	}

	rest := dsn[slash+1:]
	q := strings.Index(rest, "?")
	if q >= 0 {
		dbName = rest[:q]
		serverDSN = dsn[:slash+1] + rest[q:]
	} else {
		dbName = rest
		serverDSN = dsn[:slash+1]
	}

	if dbName == "" {
		return "", "", fmt.Errorf("invalid MYSQL_DSN: empty database name")
	}
	for _, r := range dbName {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return "", "", fmt.Errorf("invalid MYSQL_DSN: bad database name %q", dbName)
		}
	}
	return dbName, serverDSN, nil
}
