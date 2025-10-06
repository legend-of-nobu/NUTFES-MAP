-- ─────────────────────────────────────────────────────────────
-- 4) 初期データ: Index マップ（親マップ）      ※parent_map_id IS NULL
-- ─────────────────────────────────────────────────────────────
-- 画像や自然サイズは PATCH で後入れする想定のため、ここでは未設定（NULL）
-- has_floors / floor_count はサーバーサイド専管（デフォルトの 0 を使用）

INSERT INTO `maps` (`id`, `name`, `image_data`, `parent_map_id`)
VALUES
  ('e58c2b6a-6f29-41be-85b9-bdfee9a04f02', 'untitled map', NULL, NULL)
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`);