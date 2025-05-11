CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(100) NOT NULL,
    points INTEGER DEFAULT 0,
    referrer VARCHAR(36),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (referrer) REFERENCES users(id) ON DELETE SET NULL);

CREATE TABLE tasks (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    points INTEGER NOT NULL
);

CREATE TABLE user_tasks (
    user_id VARCHAR(36) NOT NULL,
    task_id VARCHAR(36) NOT NULL,
    completed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, task_id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);

-- Начальные задачи
INSERT INTO tasks (id, name, description, points) VALUES
('1', 'telegram', 'Подписаться на Telegram канал', 50),
('2', 'twitter', 'Подписаться на Twitter', 50),
('3', 'referral', 'Пригласить друга', 100),
('4', 'ad', 'Просмотр рекламы', 25);