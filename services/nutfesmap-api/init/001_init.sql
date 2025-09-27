-- 1. users テーブル
CREATE TABLE `users` (
  `id`             VARCHAR(36)       NOT NULL,
  `username`       VARCHAR(64)       NOT NULL UNIQUE,
  `password_hash`  VARCHAR(255)      NOT NULL,
  `created_at`     DATETIME(6)       NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at`     DATETIME(6)       NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY(`id`)
);

-- 6. refresh_token テーブル
CREATE TABLE refresh_tokens (
  `id`            CHAR(36)     NOT NULL,
  `user_id`       CHAR(36)     NOT NULL,
  `token_hash`    CHAR(64)     NOT NULL,
  `expires_at`    DATETIME(6)  NOT NULL,
  `revoked_at`    DATETIME(6)  NULL,
  `created_at`    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `idx_user` (`user_id`),
  CONSTRAINT `fk_rt_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;