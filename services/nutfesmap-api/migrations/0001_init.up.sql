-- schema_full_longtext_no_triggers.sql
SET NAMES utf8mb4;
SET time_zone = '+00:00';

-- ───────────────── users ─────────────────
CREATE TABLE IF NOT EXISTS `users` (
  `id`            VARCHAR(36)  NOT NULL,
  `username`      VARCHAR(64)  NOT NULL UNIQUE,
  `password_hash` VARCHAR(255) NOT NULL,
  `created_at`    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `modified_at`   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
                                      ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─────────────── refresh_tokens ───────────────
CREATE TABLE IF NOT EXISTS `refresh_tokens` (
  `id`         VARCHAR(36)  NOT NULL,
  `user_id`    VARCHAR(36)  NOT NULL,
  `token_hash` VARCHAR(64)  NOT NULL,
  `expires_at` DATETIME(6)  NOT NULL,
  `revoked_at` DATETIME(6)  NULL,
  `created_at` DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `idx_rt_user` (`user_id`),
  CONSTRAINT `fk_rt_user` FOREIGN KEY (`user_id`)
    REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─────────────────── maps ───────────────────
-- 画像base64は LONGTEXT（NULL可）。name/natural_* は非NULLで既定値あり。
CREATE TABLE IF NOT EXISTS `maps` (
  `id`             VARCHAR(36)   NOT NULL,
  `name`           VARCHAR(255)  NOT NULL DEFAULT '',
  `image_data`     LONGTEXT      NULL,               -- base64 string（NULL許可）
  `natural_width`  INT UNSIGNED  NOT NULL DEFAULT 0,
  `natural_height` INT UNSIGNED  NOT NULL DEFAULT 0,
  `parent_map_id`  VARCHAR(36)   NULL,
  `has_floors`     TINYINT(1)    NOT NULL DEFAULT 0,
  `floor_count`    INT UNSIGNED  NOT NULL DEFAULT 0,
  `created_at`     DATETIME(6)   NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `deleted_at`     DATETIME(6)   NULL,
  `modified_at`    DATETIME(6)   NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
                                        ON UPDATE CURRENT_TIMESTAMP(6),

  CONSTRAINT `pk_maps` PRIMARY KEY (`id`),

  CONSTRAINT `fk_maps_parent`
    FOREIGN KEY (`parent_map_id`) REFERENCES `maps`(`id`)
    ON DELETE CASCADE,

  CONSTRAINT `chk_maps_natural_width`  CHECK (`natural_width`  >= 0),
  CONSTRAINT `chk_maps_natural_height` CHECK (`natural_height` >= 0),
  CONSTRAINT `chk_maps_floor_count`    CHECK (`floor_count`    >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX `idx_maps_parent`      ON `maps` (`parent_map_id`);
CREATE INDEX `idx_maps_modified_at` ON `maps` (`modified_at`);

-- ─────────────────── pins ───────────────────
-- 説明画像も LONGTEXT（NULL可）
CREATE TABLE IF NOT EXISTS `pins` (
  `id`                 VARCHAR(36)  NOT NULL,
  `map_id`             VARCHAR(36)  NOT NULL,
  `name`               VARCHAR(100) NOT NULL,
  `description`        VARCHAR(1000) NULL,
  `description_image`  LONGTEXT     NULL,            -- base64 string
  `type`               ENUM('area_selector','exhibit','service','info')
                       NOT NULL DEFAULT 'exhibit',
  `link_to_map_id`     VARCHAR(36)  NULL,
  `x_norm`             DECIMAL(8,6) NOT NULL,
  `y_norm`             DECIMAL(8,6) NOT NULL,
  `category`           ENUM('food','stage','exhibition','game','service','other') NOT NULL,
  `status`             ENUM('open','paused','closed') NOT NULL DEFAULT 'open',
  `wait_minutes`       INT UNSIGNED NOT NULL DEFAULT 0,
  `created_at`         DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `modified_at`        DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
                                      ON UPDATE CURRENT_TIMESTAMP(6),

  CONSTRAINT `pk_pins` PRIMARY KEY (`id`),

  CONSTRAINT `fk_pins_map`
    FOREIGN KEY (`map_id`) REFERENCES `maps`(`id`)
    ON DELETE CASCADE,

  CONSTRAINT `fk_pins_link_to_map`
    FOREIGN KEY (`link_to_map_id`) REFERENCES `maps`(`id`)
    ON DELETE SET NULL,

  CONSTRAINT `chk_pins_x_range`     CHECK (`x_norm` >= 0.0 AND `x_norm` <= 1.0),
  CONSTRAINT `chk_pins_y_range`     CHECK (`y_norm` >= 0.0 AND `y_norm` <= 1.0),
  CONSTRAINT `chk_pins_wait_nonneg` CHECK (`wait_minutes` >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX `idx_pins_map`       ON `pins` (`map_id`);
CREATE INDEX `idx_pins_status`    ON `pins` (`status`);
CREATE INDEX `idx_pins_modified`  ON `pins` (`modified_at`);
CREATE INDEX `idx_pins_category`  ON `pins` (`category`);