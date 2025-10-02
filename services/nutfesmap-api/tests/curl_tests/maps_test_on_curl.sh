#!/usr/bin/env bash
set -euo pipefail

# ========= 共通設定（環境変数で上書き可） =========
BASE="${BASE:-http://localhost:8080}"
UA="${UA:-curl-test/1.2}"
ROOT_NAME="${ROOT_NAME:-2025 NUTFES}"   # 既存ルートを名前で優先選択
PATCH_ROOT="${PATCH_ROOT:-0}"           # 1 にすると根マップの自然サイズを最小限PATCH
B64_IMG="${B64_IMG:-iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAAWgmWQ0AAAAASUVORK5CYII=}"

need() { command -v "$1" >/dev/null 2>&1 || { echo "ERROR: '$1' not found"; exit 1; }; }
need jq
need awk
need sed

curlj() { curl -sS -A "$UA" -H 'Accept: application/json' "$@"; }

echo "BASE=$BASE"

# 1) インデックス取得
echo
echo "== 1) 親なしマップ一覧（GET /maps/index） =="
INDEX_JSON="$(curlj -X GET "$BASE/maps/index")"
echo "$INDEX_JSON" | jq

# 2) 既存のルートIDを決定（ROOT_NAME があれば優先、なければ親なしの先頭）
echo
echo "== 2) 既存のルートIDを決定 =="
ROOT_ID="$(
  echo "$INDEX_JSON" \
  | jq -er --arg NAME "$ROOT_NAME" '
      .items
      | map(select(.parentMapId == null))
      | ( (map(select(.name == $NAME))[0].id) // (.[0].id) )
    '
)"
echo "ROOT_ID=$ROOT_ID"

# 3) ルートの詳細表示（内容は変更しない）
echo
echo "== 3) ルート詳細（GET /maps/:id） =="
curlj -X GET "$BASE/maps/$ROOT_ID" | jq

# 4) （任意）画像や自然サイズを最小限で PATCH したい場合はここで実施
if [[ "$PATCH_ROOT" == "1" ]]; then
  echo
  echo "== 4) ルートを最小限 PATCH（任意） =="
  curlj -X PATCH "$BASE/maps/$ROOT_ID" \
    -H 'Content-Type: application/json' \
    -d @- <<JSON | jq
{ "naturalWidth": 2048, "naturalHeight": 1536, "imageData": "$B64_IMG" }
JSON
fi

# 5) フロアを2つ追加（POST /maps/:mapId/floors x2）
echo
echo "== 5) フロアを2つ追加（POST /maps/:mapId/floors x2） =="
F1_ID="$(curlj -X POST "$BASE/maps/$ROOT_ID/floors" | jq -er '.id')"
echo "F1_ID=$F1_ID"
F2_ID="$(curlj -X POST "$BASE/maps/$ROOT_ID/floors" | jq -er '.id')"
echo "F2_ID=$F2_ID"

# 6) フロア一覧（1F..順）
echo
echo "== 6) フロア一覧（GET /maps/:mapId/floors） 1F..順 =="
curlj -X GET "$BASE/maps/$ROOT_ID/floors" | jq

# 7) 最上階以外の削除はエラーであることを確認（/floors/1）
echo
echo "== 7) 最上階以外は削除できないことの確認（DELETE /maps/:id/floors/1） =="
set +e
HTTP_CODE="$(curl -sS -A "$UA" -H 'Accept: application/json' -X DELETE "$BASE/maps/$ROOT_ID/floors/1" -i | awk 'NR==1{print $2}')"
set -e
echo "HTTP_CODE=$HTTP_CODE  # 400 を想定"

# 8) 最上階（2F）を削除
echo
echo "== 8) 最上階を削除（DELETE /maps/:id/floors/2） =="
curl -sS -A "$UA" -H 'Accept: application/json' -X DELETE "$BASE/maps/$ROOT_ID/floors/2" -i

# 9) フロア一覧を再確認
echo
echo "== 9) フロア一覧を再確認（GET /maps/:id/floors） =="
curlj -X GET "$BASE/maps/$ROOT_ID/floors" | jq

# 10) ETag を取得して条件付き取得を試す
echo
echo "== 10) 条件付き取得（ETag） =="
ETAG="$(curl -sS -A "$UA" -i "$BASE/maps/index" | awk -F': ' 'tolower($1)=="etag"{gsub("\r","",$2);print $2;exit}')"
echo "ETag=$ETAG"
curl -sS -A "$UA" -H "If-None-Match: $ETAG" "$BASE/maps/index" -i

# 11) 片付け：作成したフロアを個別削除（ルートは削除しない）
echo
echo "== 11) 片付け：作成したフロアを個別削除（DELETE /maps/:floorId） =="
# 2F はすでに削除済みの可能性があるので失敗は無視
set +e
curl -sS -A "$UA" -H 'Accept: application/json' -X DELETE "$BASE/maps/$F2_ID" -i >/dev/null
set -e
curl -sS -A "$UA" -H 'Accept: application/json' -X DELETE "$BASE/maps/$F1_ID" -i

echo
echo "== 完了 =="
