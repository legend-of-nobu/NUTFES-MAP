# Makefile - nutfesmapDispatchService 用
COMPOSE_FILE := docker-compose.yml
DC := docker compose -f $(COMPOSE_FILE)

.PHONY: up down build rebuild pull logs ps api-sh db-sh prune watch env
.PHONY: db-wait db-rebuild migrate-rebuild db-init


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
	$(DC) exec nutfesmap-database sh || $(DC) exec nutfesmap-database bash

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

## DBコンテナをリビルドして起動（データは保持）
db-rebuild:
	$(DC) build --no-cache $(DB_SVC)
	$(DC) up -d $(DB_SVC)

## migrateコンテナをリビルド
migrate-rebuild:
	$(DC) build --no-cache $(MIGRATE_SVC)

## DB初期化（全ボリューム破棄→DBのみ再構築→ヘルス待ち→マイグレーション適用）
## データを完全に消して作り直したいときはこれ
db-init:
	@echo "⚠️  全ボリュームを破棄してDBを初期化する（他サービスのデータも消える）"
	@read -p "続行しますか？ [y/N] " ans; \
	if [ "$$ans" != "y" ] && [ "$$ans" != "Y" ]; then echo "中止"; exit 1; fi
	# 1) すべて停止 & ボリューム破棄（DBデータを空にする）
	$(DC) down -v
	# 2) DBだけ先にビルド・起動
	$(DC) up -d --build $(DB_SVC)
	# 3) DBのヘルスチェックを待つ
	$(MAKE) db-wait
	# 4) スキーマ適用（golang-migrate）
	$(MAKE) migrate-up
	@echo "✅ DB 初期化完了"

## DBが healthy になるまで待つ（healthcheck 必須）
db-wait:
	@echo "⏳ Waiting for $(DB_SVC) to be healthy..."
	@cid=$$($(DC) ps -q $(DB_SVC)); \
	if [ -z "$$cid" ]; then echo "❌ $(DB_SVC) が起動していません"; exit 1; fi; \
	while :; do \
		status=$$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{end}}' $$cid 2>/dev/null || true); \
		[ "$$status" = "healthy" ] && break; \
		[ "$$status" = "unhealthy" ] && echo "❌ DB unhealthy"; \
		[ "$$status" = "unhealthy" ] && exit 1; \
		printf "."; sleep 1; \
	done; \
	echo " OK"