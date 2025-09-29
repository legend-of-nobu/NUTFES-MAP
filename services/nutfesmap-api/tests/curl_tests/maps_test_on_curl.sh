#!/usr/bin/env bash
set -euo pipefail

# ========= 共通設定 =========
BASE="${BASE:-http://localhost:8080}"
UA="${UA:-curl-test/1.0}"

# 画像のダミー base64（必要に応じて差し替え）
B64_IMG='iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAAWgmWQ0AAAAASUVORK5CYII='

curlj() {
  curl -sS -A "$UA" -H 'Accept: application/json' "$@"
}

echo "== 1) 親なしマップ一覧（GET /maps/index） =="
curlj -X GET "$BASE/maps/index" | jq

echo
echo "== 2) 既存の親マップ ID を取得（シード済み想定） =="
# 名前（例: 2025 NUTFES）で探し、無ければ最初の親マップを使う。どちらも無ければエラー終了。
MAP_ID="$(
  curlj -X GET "$BASE/maps/index" \
  | jq -er '
      .items
      | map(select(.parentMapId == null))          # 親なし = ルート
      | ( map(select(.name == "2025 NUTFES"))[0].id
          // (.[0].id)
        )'
)"
echo "MAP_ID=$MAP_ID"

echo
echo "== 3) 親マップの詳細（GET /maps/:id） =="
curlj -X GET "$BASE/maps/$MAP_ID" | jq

echo
echo "== 4) 親マップの内容を投入（PATCH /maps/:id）"
echo "   - 編集可能: name, imageData(base64), naturalWidth, naturalHeight, parentMapId"
echo "   - hasFloors/floorCount はサーバ専管（送らない）"
curlj -X PATCH "$BASE/maps/$MAP_ID" \
  -H 'Content-Type: application/json' \
  -d @- <<JSON | jq
{
  "name": "2025 技大祭メインマップ",
  "imageData": "$B64_IMG",
  "naturalWidth": 4096,
  "naturalHeight": 3072
}
JSON

echo
echo "== 5) 子マップの作成（親にぶら下げる空マップを作成 → PATCHで内容投入） =="
CHILD_MAP_ID="$(
  curlj -X POST "$BASE/maps" \
    -H 'Content-Type: application/json' \
    -d @- <<JSON \
  | jq -er '.id'
{ "parentMapId": "$MAP_ID" }
JSON
)"
echo "CHILD_MAP_ID=$CHILD_MAP_ID"

echo
echo "== 5-1) 子マップへ内容投入（PATCH /maps/:id） =="
curlj -X PATCH "$BASE/maps/$CHILD_MAP_ID" \
  -H 'Content-Type: application/json' \
  -d @- <<JSON | jq
{
  "name": "本館 1F",
  "imageData": "$B64_IMG",
  "naturalWidth": 2048,
  "naturalHeight": 1536
}
JSON

echo
echo "== 6) 親の付け替え / 解除（PATCH /maps/:id） =="
echo "   - 子の親解除（ルートへ移動）"
curlj -X PATCH "$BASE/maps/$CHILD_MAP_ID" \
  -H 'Content-Type: application/json' \
  -d '{"parentMapId": null}' | jq

echo "   - 親を再設定（元の親へ戻す）"
curlj -X PATCH "$BASE/maps/$CHILD_MAP_ID" \
  -H 'Content-Type: application/json' \
  -d @- <<JSON | jq
{ "parentMapId": "$MAP_ID" }
JSON

echo
echo "== 7) 親マップの自然サイズだけ更新（必要時のみ） =="
curlj -X PATCH "$BASE/maps/$MAP_ID" \
  -H 'Content-Type: application/json' \
  -d '{"naturalWidth": 4320, "naturalHeight": 3240}' | jq

echo
echo "== 8) 親マップ詳細（子メタ含む）確認 =="
curlj -X GET "$BASE/maps/$MAP_ID" | jq

echo
echo "== 9) 子マップ削除（配下にピン等あれば再帰で削除） =="
# ※ シード済みの親は残すため、子のみ削除
curl -sS -A "$UA" -X DELETE "$BASE/maps/$CHILD_MAP_ID" -i

echo
echo "== 10) 条件付き取得（ETag） =="
ETAG="$(curl -sS -A "$UA" -i "$BASE/maps/index" | awk -F': ' 'tolower($1)=="etag"{gsub("\r","",$2);print $2;exit}')"
echo "ETag=$ETAG"
curl -sS -A "$UA" -H "If-None-Match: $ETAG" "$BASE/maps/index" -i
