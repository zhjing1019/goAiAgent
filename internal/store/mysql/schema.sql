-- 第 5 步：MySQL 表结构（也可由程序 Migrate 自动创建）
-- 使用：mysql -u root -p go_agent < internal/store/mysql/schema.sql

-- CREATE DATABASE：创建数据库
-- IF NOT EXISTS：如果不存在才创建（避免重复创建报错）
-- go_agent：数据库名字（你的 AI 对话项目数据库）
-- DEFAULT CHARSET utf8mb4：支持所有中文 + emoji 表情
-- COLLATE utf8mb4_unicode_ci：排序规则（不区分大小写）

CREATE DATABASE IF NOT EXISTS go_agent DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE go_agent;

CREATE TABLE IF NOT EXISTS sessions (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    title      VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS messages (
    id           BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    session_id   VARCHAR(36)  NOT NULL,
    seq          INT          NOT NULL COMMENT '会话内顺序，从 1 开始',
    role         VARCHAR(20)  NOT NULL,
    content      MEDIUMTEXT,
    tool_calls   JSON         NULL COMMENT 'assistant 的 tool_calls',
    tool_call_id VARCHAR(64)  NULL,
    tool_name    VARCHAR(64)  NULL,
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_messages_session (session_id, seq),
    CONSTRAINT fk_messages_session FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
