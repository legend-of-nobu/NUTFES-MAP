#!/usr/bin/env bash
set -euo pipefail

# ========= 共通設定 =========
BASE="${BASE:-http://localhost:8080}"
UA="${UA:-curl-test/1.5}"
B64_IMG="${B64_IMG:-iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAAWgmWQ0AAAAASUVORK5CYII=}"
CLEANUP="${CLEANUP:-1}"  # 1=最後に削除、0=残す

need() { command -v "$1" >/dev/null 2>&1 || { echo "ERROR: '$1' not found"; exit 1; }; }
need jq
need curl

curlj() { curl -sS -A "$UA" -H 'Accept: application/json' "$@"; }
say() { printf '\n== %s ==\n' "$*"; }
fail() { echo "ERROR: $*"; exit 1; }

assert_eq() {
  local got="$1" want="$2" msg="${3:-}"
  if [ "$got" != "$want" ]; then
    fail "ASSERT FAIL: $msg (got=$got want=$want)"
  fi
}

create_root_map() {
  local name="$1"
  local id
  id="$(
    curlj -X POST "$BASE/maps" \
      -H 'Content-Type: application/json' \
      -d '{"parentMapId":null}' \
    | jq -er '.id'
  )" || fail "root map を作成できませんでした（$name）"

  # 名前・画像など最低限を PATCH
  curlj -X PATCH "$BASE/maps/$id" \
    -H 'Content-Type: application/json' \
    -d @- >/dev/null <<JSON
{ "name":"$name", "imageData":"$B64_IMG", "naturalWidth":2048, "naturalHeight":1536 }
JSON
  echo "$id"
}

add_floors() {
  local root_id="$1"
  local n="$2"
  local i fid
  i=1
  while [ "$i" -le "$n" ]; do
    fid="$(curlj -X POST "$BASE/maps/$root_id/floors" | jq -er '.id')" \
      || fail "floor の作成に失敗しました（root=$root_id, i=$i）"
    curlj -X PATCH "$BASE/maps/$fid" \
      -H 'Content-Type: application/json' \
      -d "{\"name\":\"${i}F\"}" >/dev/null
    i=$((i+1))
  done
}

# ----- ツリー構築ユーティリティ -----

get_map() { curlj -X GET "$BASE/maps/$1"; }

# 指定 ID を根とする部分木 JSON を出力:
# {id,name,floorCount,hasFloors,naturalWidth,naturalHeight,children:[ ... ]}
build_tree() {
  local id="$1"
  local map_json children_ids children_json cj

  map_json="$(get_map "$id")" || return 1

  # 子 ID の列（改行区切り）を作る
  if [ "$(echo "$map_json" | jq -r '.floorCount')" != "0" ]; then
    children_ids="$(curlj -X GET "$BASE/maps/$id/floors" | jq -r '.items[].map.id')"
  else
    children_ids="$(echo "$map_json" | jq -r '.children[].id')"
  fi

  children_json='[]'
  if [ -n "${children_ids:-}" ]; then
    # 改行区切りの ID 群をループ
    while IFS= read -r cid; do
      [ -z "$cid" ] && continue
      cj="$(build_tree "$cid")" || return 1
      children_json="$(jq -c --argjson child "$cj" '. + [ $child ]' <<< "$children_json")"
    done <<< "$children_ids"
  fi

  # ノードに children を差し込んで返す
  echo "$map_json" | jq -c --argjson children "$children_json" \
    '{id,name,floorCount,hasFloors,naturalWidth,naturalHeight,children:$children}'
}

# すべての root（parent_map_id=NULL）を列挙してツリーを配列で出力
print_full_tree_from_roots() {
  local idx root_ids_json rid subtree trees
  idx="$(curlj -X GET "$BASE/maps/index")" || return 1
  trees='[]'

  # ルートの ID 群を 1 行ずつ取り出す
  while IFS= read -r rid; do
    [ -z "$rid" ] && continue
    subtree="$(build_tree "$rid")" || return 1
    trees="$(jq -c --argjson child "$subtree" '. + [ $child ]' <<< "$trees")"
  done < <(echo "$idx" | jq -r '.items[].id')

  echo "$trees"
}

# ========== テスト本体 ==========

echo "BASE=$BASE"

say "1) Index（参考表示）"
curlj -X GET "$BASE/maps/index" | jq

say "2) 3つの root マップを作成（Index 直下＝root として作成）"
LECTURE_ID="$(create_root_map "講義棟エリア")"
OUTDOOR_ID="$(create_root_map "屋外エリア")"
KITCHEN_ID="$(create_root_map "キッチンカーエリア")"
echo "LECTURE_ID=$LECTURE_ID"
echo "OUTDOOR_ID=$OUTDOOR_ID"
echo "KITCHEN_ID=$KITCHEN_ID"

say "3) 講義棟エリアへ 1F..3F を追加"
add_floors "$LECTURE_ID" 3

say "4) フロアスタック確認（講義棟エリア）"
STACK_JSON="$(curlj -X GET "$BASE/maps/$LECTURE_ID/floors")"
echo "$STACK_JSON" | jq
FCOUNT="$(echo "$STACK_JSON" | jq -r '.floorCount')"
assert_eq "$FCOUNT" "3" "floorCount must be 3"
NAMES="$(echo "$STACK_JSON" | jq -r '.items[].map.name' | paste -sd',' -)"
assert_eq "$NAMES" "1F,2F,3F" "floor names must be 1F,2F,3F"

say "5) Index を再取得（作成結果の確認用）"
IDX_JSON="$(curlj -X GET "$BASE/maps/index")"
echo "$IDX_JSON" | jq '.items | map({id,name,floorCount})'

# jq 1.5 でも動くよう any() は使わず長さ判定で確認
for want in "講義棟エリア" "屋外エリア" "キッチンカーエリア"; do
  echo "$IDX_JSON" | jq -e --arg w "$want" \
    '.items | map(select(.name==$w)) | length > 0' >/dev/null \
    || fail "Index に $want が見つかりません"
done
echo "$IDX_JSON" | jq -e --arg w "講義棟エリア" \
  '.items | map(select(.name==$w and .floorCount==3)) | length > 0' >/dev/null \
  || fail "Index 上の '講義棟エリア' の floorCount が 3 ではありません"

say "6) ルートからの JSON ツリーを出力（全マップ）"
FULL_TREE="$(print_full_tree_from_roots)"
echo "$FULL_TREE" | jq

if [ "${CLEANUP}" = "1" ]; then
  say "7) 片付け：今回作成した 3 root を削除（再帰）"
  for id in "$LECTURE_ID" "$OUTDOOR_ID" "$KITCHEN_ID"; do
    echo "DELETE /maps/$id"
    curl -sS -A "$UA" -H 'Accept: application/json' -X DELETE "$BASE/maps/$id" -i | head -n1
  done
else
  say "7) CLEANUP=0 のため削除をスキップしました"
fi

say "完了"
