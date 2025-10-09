# Makefile - nutfesmapDispatchService 用
COMPOSE_FILE := docker-compose.yml
DC := docker compose -f $(COMPOSE_FILE)

.PHONY: up down build rebuild pull logs ps api-sh db-sh prune watch env
.PHONY: db-rebuild migrate-rebuild db-init


## 起動（全サービス）
up:
	$(DC) up -d

## 停止
down:
	$(DC) down

## APIのみビルドして起動
build:
	$(DC) build nutfesmap-api
	$(DC) up -d nutfesmap-api

## 全ビルドし直して起動（キャッシュなし）
rebuild:
	$(DC) down -v
	$(DC) up -d --build

## イメージの pull（依存イメージの更新）
pull:
	$(DC) pull

## ログ監視（全サービス）
logs:
	$(DC) logs -f

## コンテナ状態一覧
ps:
	$(DC) ps

## nutfesmap-api コンテナにシェルログイン
api-sh:
	$(DC) exec nutfesmap-api sh

## nutfesmap-database コンテナにログイン（MySQL入り Alpine/Ubuntu に対応）
db-sh:
	$(DC) exec nutfesmap-database mysql -uwebapp -pPassword

## 未使用ネットワークやボリューム削除
prune:
	docker system prune -f

## .env 中身を確認（API用）
env:
	cat services/nutfesmap-api/config/.env

## Docker Compose v2.22〜 の hot-reload
watch:
	@version=$$(docker compose version --short); \
	if [ "$$(printf '%s\n' "2.22.0" "$$version" | sort -V | head -n1)" = "2.22.0" ]; then \
		COMPOSE_FILE=$(COMPOSE_FILE) docker compose watch; \
	else \
		echo "❌ 'docker compose watch' には v2.22.0 以上が必要です（現在: $$version）"; \
	fi

db-rebuild:
	$(DC) build --no-cache $(DB_SVC)
	$(DC) up -d $(DB_SVC)

migrate-rebuild:
	$(DC) build --no-cache $(MIGRATE_SVC)

	# ==== デフォルト（未設定なら使われます）====
API_SVC     ?= nutfesmap-api
DB_SVC      ?= nutfesmap-database
MIGRATE_SVC ?= nutfesmap-migrate
DB_WAIT_TIMEOUT ?= 120

# ==== migrate 用ターゲット ====
.PHONY: migrate-up migrate-down migrate-down-all migrate-force migrate-create migrate-sh

## 最新まで適用
migrate-up:
	$(DC) run --rm -e MIGRATE_CMD="up" $(MIGRATE_SVC)

## 1つ戻す
migrate-down:
	$(DC) run --rm -e MIGRATE_CMD="down 1" $(MIGRATE_SVC)

## 全部ロールバック
migrate-down-all:
	$(DC) run --rm -e MIGRATE_CMD="down -all" $(MIGRATE_SVC)

## 強制バージョン固定（使い方: make migrate-force VERSION=202510070001）
migrate-force:
	@test -n "$(VERSION)" || (echo "Usage: make migrate-force VERSION=YYYYMMDDHHMM"; exit 1)
	$(DC) run --rm -e MIGRATE_CMD="force $(VERSION)" $(MIGRATE_SVC)

## 新規マイグレーション作成（使い方: make migrate-create NAME=add_users_table）
migrate-create:
	@test -n "$(NAME)" || (echo "Usage: make migrate-create NAME=snake_case_name"; exit 1)
	$(DC) run --rm -e MIGRATE_CMD="create" -e CREATE_NAME="$(NAME)" $(MIGRATE_SVC)

## シェルで入る
migrate-sh:
	$(DC) run --rm $(MIGRATE_SVC) sh || $(DC) run --rm $(MIGRATE_SVC) bash

## DB初期化（全ボリューム破棄→DB再作成→待機→migrate up）
db-init:
	@echo "⚠️ 全ボリュームを破棄してDBを初期化します（他サービスのデータも消えます）"
	@read -p "続行しますか？ [y/N] " ans; \
	if [ "$$ans" != "y" ] && [ "$$ans" != "Y" ]; then echo "中止"; exit 1; fi
	$(DC) down -v
	$(DC) up -d --build $(DB_SVC)
	# ← db-wait を呼ばない
	$(MAKE) migrate-up
	@echo "✅ DB 初期化完了"
	$(DC) up -d

	# Makefile（追記）

DEV_FILES  := -f docker-compose.yml
PROD_FILES := -f docker-compose.yml -f docker-compose.prod.yml

.PHONY: up-dev down-dev logs-dev ps-dev rebuild-dev pull-dev
.PHONY: up-prod down-prod logs-prod ps-prod rebuild-prod pull-prod
.PHONY: web-prod api-prod docs-prod tunnel-prod

up-dev:
	docker compose --env-file .env.dev $(DEV_FILES) up -d
down-dev:
	docker compose --env-file .env.dev $(DEV_FILES) down
logs-dev:
	docker compose --env-file .env.dev $(DEV_FILES) logs -f
ps-dev:
	docker compose --env-file .env.dev $(DEV_FILES) ps
rebuild-dev:
	docker compose --env-file .env.dev $(DEV_FILES) down -v
	docker compose --env-file .env.dev $(DEV_FILES) up -d --build
pull-dev:
	docker compose --env-file .env.dev $(DEV_FILES) pull

up-prod:
	docker compose --env-file .env.prod $(PROD_FILES) up -d
down-prod:
	docker compose --env-file .env.prod $(PROD_FILES) down
logs-prod:
	docker compose --env-file .env.prod $(PROD_FILES) logs -f
ps-prod:
	docker compose --env-file .env.prod $(PROD_FILES) ps
rebuild-prod:
	docker compose --env-file .env.prod $(PROD_FILES) down -v
	docker compose --env-file .env.prod $(PROD_FILES) up -d --build
pull-prod:
	docker compose --env-file .env.prod $(PROD_FILES) pull

web-prod:
	docker compose --env-file .env.prod $(PROD_FILES) up -d nutfesmap-web
api-prod:
	docker compose --env-file .env.prod $(PROD_FILES) up -d nutfesmap-api
docs-prod:
	docker compose --env-file .env.prod $(PROD_FILES) up -d nutfesmap-api-document
tunnel-prod:
	docker compose --env-file .env.prod $(PROD_FILES) up -d cloudflared