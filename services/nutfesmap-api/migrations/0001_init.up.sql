-- 文字コード・照合順序の推奨（必要に応じてスキーマ単位で設定）
-- CREATE DATABASE nutfesmap CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

SET NAMES utf8mb4;
SET time_zone = '+00:00';

-- ─────────────────────────────────────────────────────────────
-- 0) 既存テーブル（そのまま利用）
-- ─────────────────────────────────────────────────────────────
-- users / refresh_tokens は質問にある定義を前提にする。
-- 必要ならこのまま再掲して実行してもよい。

-- users
CREATE TABLE IF NOT EXISTS `users` (
  `id`             CHAR(36)        NOT NULL,
  `username`       VARCHAR(64)     NOT NULL UNIQUE,
  `password_hash`  VARCHAR(255)    NOT NULL,
  `created_at`     DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at`     DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- refresh_tokens
CREATE TABLE IF NOT EXISTS `refresh_tokens` (
  `id`            CHAR(36)     NOT NULL,
  `user_id`       CHAR(36)     NOT NULL,
  `token_hash`    CHAR(64)     NOT NULL,
  `expires_at`    DATETIME(6)  NOT NULL,
  `revoked_at`    DATETIME(6)  NULL,
  `created_at`    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `idx_rt_user` (`user_id`),
  CONSTRAINT `fk_rt_user` FOREIGN KEY (`user_id`)
    REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─────────────────────────────────────────────────────────────
-- 1) maps: 階層マップ（背景画像メタ含む）
-- ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS `maps` (
  `id`              CHAR(36)      NOT NULL,
  `name`            VARCHAR(255)  NOT NULL,
  -- 背景画像（生バイナリ）。API 層で base64 に変換して返す。
  `image_blob`      LONGBLOB      NOT NULL,
  `natural_width`   INT UNSIGNED  NOT NULL,
  `natural_height`  INT UNSIGNED  NOT NULL,
  `parent_map_id`   CHAR(36)      NULL,
  `has_floors`      TINYINT(1)    NOT NULL DEFAULT 0,
  `floor_count`     INT UNSIGNED  NOT NULL DEFAULT 0,
  `created_at`      DATETIME(6)   NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at`      DATETIME(6)   NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

  CONSTRAINT `pk_maps` PRIMARY KEY (`id`),

  CONSTRAINT `fk_maps_parent`
    FOREIGN KEY (`parent_map_id`) REFERENCES `maps`(`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 表示・検索用インデックス
CREATE INDEX `idx_maps_parent` ON `maps` (`parent_map_id`);
CREATE INDEX `idx_maps_updated_at` ON `maps` (`updated_at`);

-- 値域制約（MySQL 8.0 では CHECK 有効）
ALTER TABLE `maps`
  ADD CONSTRAINT `chk_maps_natural_width`  CHECK (`natural_width`  >= 1),
  ADD CONSTRAINT `chk_maps_natural_height` CHECK (`natural_height` >= 1),
  ADD CONSTRAINT `chk_maps_floor_count`    CHECK (`floor_count` >= 0);

-- ─────────────────────────────────────────────────────────────
-- 2) pins: マップ上のピン（企画・サービス・エリア遷移など）
-- ─────────────────────────────────────────────────────────────
-- ENUM は OpenAPI の定義に合わせる
DROP TABLE IF EXISTS `pins`;
CREATE TABLE `pins` (
  `id`                   CHAR(36)       NOT NULL,
  `map_id`               CHAR(36)       NOT NULL,
  `name`                 VARCHAR(100)   NOT NULL,
  `description`          VARCHAR(1000)  NULL,
  `description_image`    LONGBLOB       NULL,  -- 説明用画像（生バイナリ）
  `type`                 ENUM('area_selector','exhibit','service','info') DEFAULT 'exhibit' NOT NULL,
  `link_to_map_id`       CHAR(36)       NULL,  -- area_selector 用の遷移先
  `x_norm`               DECIMAL(8,6)   NOT NULL,  -- 0.000000〜1.000000 を想定
  `y_norm`               DECIMAL(8,6)   NOT NULL,
  `category`             ENUM('food','stage','exhibition','game','service','other') NOT NULL,
  `status`               ENUM('open','paused','closed') NOT NULL DEFAULT 'open',
  `wait_minutes`         INT UNSIGNED   NOT NULL DEFAULT 0,
  `created_at`           DATETIME(6)    NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at`           DATETIME(6)    NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

  CONSTRAINT `pk_pins` PRIMARY KEY (`id`),

  CONSTRAINT `fk_pins_map`
    FOREIGN KEY (`map_id`) REFERENCES `maps`(`id`)
    ON DELETE CASCADE,

  -- 遷移先マップが削除されたらリンクのみ切る（ピンは残す）
  CONSTRAINT `fk_pins_link_to_map`
    FOREIGN KEY (`link_to_map_id`) REFERENCES `maps`(`id`)
    ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 表示・検索用インデックス
CREATE INDEX `idx_pins_map`      ON `pins` (`map_id`);
CREATE INDEX `idx_pins_status`   ON `pins` (`status`);
CREATE INDEX `idx_pins_updated`  ON `pins` (`updated_at`);
CREATE INDEX `idx_pins_category` ON `pins` (`category`);

-- 値域チェック（0〜1）
ALTER TABLE `pins`
  ADD CONSTRAINT `chk_pins_x_range` CHECK (`x_norm` >= 0.0 AND `x_norm` <= 1.0),
  ADD CONSTRAINT `chk_pins_y_range` CHECK (`y_norm` >= 0.0 AND `y_norm` <= 1.0);

-- 待ち時間は 0 以上
ALTER TABLE `pins`
  ADD CONSTRAINT `chk_pins_wait_nonneg` CHECK (`wait_minutes` >= 0);

-- ─────────────────────────────────────────────────────────────
-- 3) pin_waittime_logs: 待ち時間の更新履歴（監査・差分配信の基礎）
-- ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS `pin_waittime_logs` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `pin_id`        CHAR(36)        NOT NULL,
  `old_minutes`   INT UNSIGNED    NOT NULL,
  `new_minutes`   INT UNSIGNED    NOT NULL,
  `note`          VARCHAR(200)    NULL,           -- API の note を保存
  `changed_by`    CHAR(36)        NULL,           -- 将来の認証導入に備えて（現仕様では NULL 可）
  `created_at`    DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

  CONSTRAINT `pk_pin_waittime_logs` PRIMARY KEY (`id`),

  CONSTRAINT `fk_pwlog_pin`
    FOREIGN KEY (`pin_id`) REFERENCES `pins`(`id`)
    ON DELETE CASCADE,

  CONSTRAINT `fk_pwlog_user`
    FOREIGN KEY (`changed_by`) REFERENCES `users`(`id`)
    ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX `idx_pwlog_pin_created` ON `pin_waittime_logs` (`pin_id`, `created_at`);
