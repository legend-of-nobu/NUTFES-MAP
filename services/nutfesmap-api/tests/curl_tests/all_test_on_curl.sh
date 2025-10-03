#!/usr/bin/env bash
# maps_and_pins_smoke.sh — マップ作成＋ピン作成/検証/削除の統合テスト
set -euo pipefail
export LC_ALL=C

# ========= 共通設定 =========
BASE="${BASE:-http://localhost:8080}"
UA="${UA:-curl-test/maps-pins/1.0}"
B64_IMG="${B64_IMG:-iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAAWgmWQ0AAAAASUVORK5CYII=}"
CLEANUP="${CLEANUP:-1}"   # 1=最後に削除、0=残す

need() { command -v "$1" >/dev/null 2>&1 || { echo "ERROR: '$1' not found"; exit 1; }; }
need jq
need curl

curlj() { curl -sS -A "$UA" -H 'Accept: application/json' "$@"; }
curlc() { curl -sS -A "$UA" -H 'Accept: application/json' -o /dev/null -w "%{http_code}" "$@"; }

say()  { printf '\n== %s ==\n' "$*"; }
fail() { echo "ERROR: $*"; exit 1; }

assert_eq() {
  local got="$1" want="$2" msg="${3:-}"
  [ "$got" = "$want" ] || fail "ASSERT FAIL: $msg (got=$got want=$want)"
}
assert_nonempty() {
  local got="$1" msg="${2:-}"
  [ -n "$got" ] || fail "ASSERT FAIL: empty ($msg)"
}

# ========= マップ作成ユーティリティ =========

create_root_map() {
  local name="$1"
  local id
  id="$(
    curlj -X POST "$BASE/maps" \
      -H 'Content-Type: application/json' \
      -d '{"parentMapId":null}' \
    | jq -er '.id'
  )" || fail "root map を作成できませんでした（$name）"

  # 最低限のメタ設定
  curlj -X PATCH "$BASE/maps/$id" \
    -H 'Content-Type: application/json' \
    -d @- >/dev/null <<JSON
{ "name":"$name", "imageData":"$B64_IMG", "naturalWidth":2048, "naturalHeight":1536 }
JSON
  echo "$id"
}

add_floors() {
  local root_id="$1" n="$2"
  local i=1 fid
  while [ "$i" -le "$n" ]; do
    fid="$(curlj -X POST "$BASE/maps/$root_id/floors" | jq -er '.id')" \
      || fail "floor の作成に失敗しました（root=$root_id, i=$i）"
    curlj -X PATCH "$BASE/maps/$fid" \
      -H 'Content-Type: application/json' \
      -d "{\"name\":\"${i}F\"}" >/dev/null
    i=$((i+1))
  done
}

# ========= ピン操作ユーティリティ =========

create_pin() { # returns JSON
  local map_id="$1"; shift
  local payload="$*"
  curlj -X POST "$BASE/maps/$map_id/pins" -H 'Content-Type: application/json' -d "$payload"
}

list_pins() { # returns JSON array
  local map_id="$1"
  curlj -X GET "$BASE/maps/$map_id/pins"
}

show_pin() {  # returns JSON
  local pin_id="$1"
  curlj -X GET "$BASE/pins/$pin_id"
}

update_pin() { # returns JSON
  local pin_id="$1" payload="$2"
  curlj -X PATCH "$BASE/pins/$pin_id" -H 'Content-Type: application/json' -d "$payload"
}

delete_pin() { # asserts 204
  local pin_id="$1"
  local code; code="$(curlc -X DELETE "$BASE/pins/$pin_id")"
  assert_eq "$code" "204" "delete pin ($pin_id)"
}

# ========= ルート一覧→ツリー出力（参考） =========

get_map() { curlj -X GET "$BASE/maps/$1"; }

build_tree() {
  local id="$1"
  local map_json children_ids children_json cj
  map_json="$(get_map "$id")" || return 1

  if [ "$(echo "$map_json" | jq -r '.floorCount')" != "0" ]; then
    children_ids="$(curlj -X GET "$BASE/maps/$id/floors" | jq -r '.items[].map.id')"
  else
    children_ids="$(echo "$map_json" | jq -r '.children[].id')"
  fi

  children_json='[]'
  if [ -n "${children_ids:-}" ]; then
    while IFS= read -r cid; do
      [ -z "$cid" ] && continue
      cj="$(build_tree "$cid")" || return 1
      children_json="$(jq -c --argjson child "$cj" '. + [ $child ]' <<< "$children_json")"
    done <<< "$children_ids"
  fi

  echo "$map_json" | jq -c --argjson children "$children_json" \
    '{id,name,floorCount,hasFloors,naturalWidth,naturalHeight,children:$children}'
}

print_full_tree_from_roots() {
  local idx trees='[]'
  idx="$(curlj -X GET "$BASE/maps/index")" || return 1
  while IFS= read -r rid; do
    [ -z "$rid" ] && continue
    subtree="$(build_tree "$rid")" || return 1
    trees="$(jq -c --argjson child "$subtree" '. + [ $child ]' <<< "$trees")"
  done < <(echo "$idx" | jq -r '.items[].id')
  echo "$trees"
}

# ========= テスト本体 =========

echo "BASE=$BASE"

say "1) Index（参考表示）"
curlj -X GET "$BASE/maps/index" | jq

say "2) 3つの root マップを作成（Index 直下＝root）"
LECTURE_ID="$(create_root_map "講義棟エリア")"
OUTDOOR_ID="$(create_root_map "屋外エリア")"           # linkToMapId 用にも使う
KITCHEN_ID="$(create_root_map "キッチンカーエリア")"
echo "LECTURE_ID=$LECTURE_ID"
echo "OUTDOOR_ID=$OUTDOOR_ID"
echo "KITCHEN_ID=$KITCHEN_ID"

say "3) 講義棟エリアに 1F..3F を追加"
add_floors "$LECTURE_ID" 3

say "4) フロアスタック確認（講義棟エリア）→ 1F の ID を取得"
STACK_JSON="$(curlj -X GET "$BASE/maps/$LECTURE_ID/floors")"
echo "$STACK_JSON" | jq
FCOUNT="$(echo "$STACK_JSON" | jq -r '.floorCount')"
assert_eq "$FCOUNT" "3" "floorCount must be 3"
NAMES="$(echo "$STACK_JSON" | jq -r '.items[].map.name' | paste -sd',' -)"
assert_eq "$NAMES" "1F,2F,3F" "floor names must be 1F,2F,3F"
F1_ID="$(echo "$STACK_JSON" | jq -r '.items[] | select(.map.name=="1F") | .map.id')"
assert_nonempty "$F1_ID" "F1_ID"
echo "F1_ID=$F1_ID"

# ===== ここからピンのテストを混在させる =====

say "5) 1F にピンを3件作成（Zulu/Alpha/Mike）"
PIN_A_JSON="$(create_pin "$F1_ID" '{"name":"Zulu","xNorm":0.6,"yNorm":0.1,"category":"other","type":"exhibit","status":"open"}')"
PIN_B_JSON="$(create_pin "$F1_ID" "{\"name\":\"Alpha\",\"xNorm\":0.2,\"yNorm\":0.3,\"category\":\"service\",\"type\":\"info\",\"status\":\"paused\",\"waitMinutes\":3}")"
PIN_C_JSON="$(create_pin "$F1_ID" "{\"name\":\"Mike\",\"xNorm\":0.4,\"yNorm\":0.8,\"category\":\"food\",\"type\":\"area_selector\",\"linkToMapId\":\"$OUTDOOR_ID\",\"description\":\"屋外への誘導\",\"descriptionImageData\":\"$B64_IMG\"}")"

PIN_A_ID="$(echo "$PIN_A_JSON" | jq -r '.id')"
PIN_B_ID="$(echo "$PIN_B_JSON" | jq -r '.id')"
PIN_C_ID="$(echo "$PIN_C_JSON" | jq -r '.id')"
assert_nonempty "$PIN_A_ID" "PIN_A_ID"
assert_nonempty "$PIN_B_ID" "PIN_B_ID"
assert_nonempty "$PIN_C_ID" "PIN_C_ID"
echo "PIN_A_ID=$PIN_A_ID"
echo "PIN_B_ID=$PIN_B_ID"
echo "PIN_C_ID=$PIN_C_ID"

say "6) 一覧の並び（Name 昇順）"
LIST_JSON="$(list_pins "$F1_ID")"
echo "$LIST_JSON" | jq
NAMES2="$(echo "$LIST_JSON" | jq -r '.[].name' | paste -sd',' -)"
assert_eq "$NAMES2" "Alpha,Mike,Zulu" "pins should be sorted by name asc"

say "7) 単体取得（Alpha）→ 期待値検証"
SHOW_B="$(show_pin "$PIN_B_ID")"
echo "$SHOW_B" | jq
assert_eq "$(echo "$SHOW_B" | jq -r '.name')" "Alpha" "show Alpha"

say "8) 部分更新（Alpha → Alpha改、desc=null、xNorm=0.5、status=closed、wait=10）"
UPDATED_B="$(update_pin "$PIN_B_ID" '{"name":"Alpha改","description":null,"xNorm":0.5,"status":"closed","waitMinutes":10}')"
echo "$UPDATED_B" | jq
assert_eq "$(echo "$UPDATED_B" | jq -r '.name')" "Alpha改"
assert_eq "$(echo "$UPDATED_B" | jq -r '.status')" "closed"
assert_eq "$(echo "$UPDATED_B" | jq -r '.waitMinutes')" "10"
assert_eq "$(echo "$UPDATED_B" | jq -r '.xNorm')" "0.5"
# description は omitempty の可能性があるため、存在しなければ pass
DESC_OK="$(echo "$UPDATED_B" | jq -r 'if has("description") then (.description==null) else true end')"
assert_eq "$DESC_OK" "true" "description cleared (missing or null acceptable)"

say "9) 異常系（400 を期待）"
assert_eq "$(curlc -H 'Content-Type: application/json' -X PATCH "$BASE/pins/$PIN_A_ID" -d '{"xNorm":1.2}')" "400" "xNorm out of range"
assert_eq "$(curlc -H 'Content-Type: application/json' -X PATCH "$BASE/pins/$PIN_A_ID" -d '{"type":"invalid"}')" "400" "invalid type"
assert_eq "$(curlc -H 'Content-Type: application/json' -X PATCH "$BASE/pins/$PIN_A_ID" -d '{"status":"maybe"}')" "400" "invalid status"

say "10) linkToMapId の付与→解除"
WITH_LINK="$(update_pin "$PIN_A_ID" "{\"linkToMapId\":\"$OUTDOOR_ID\"}")"
echo "$WITH_LINK" | jq
assert_eq "$(echo "$WITH_LINK" | jq -r '.linkToMapId')" "$OUTDOOR_ID"
NO_LINK="$(update_pin "$PIN_A_ID" '{"linkToMapId":null}')"
echo "$NO_LINK" | jq
LINK_OK="$(echo "$NO_LINK" | jq -r 'if has("linkToMapId") then (.linkToMapId==null) else true end')"
assert_eq "$LINK_OK" "true" "linkToMapId cleared (missing or null acceptable)"

# ===== ツリー表示（参考） =====
say "11) ルートからの JSON ツリーを出力（全マップ）"
FULL_TREE="$(print_full_tree_from_roots)"
echo "$FULL_TREE" | jq

# ===== 後始末 =====
if [ "${CLEANUP}" = "1" ]; then
  say "12) ピン削除 → マップ再帰削除"
  delete_pin "$PIN_A_ID"
  delete_pin "$PIN_B_ID"
  delete_pin "$PIN_C_ID"
  for id in "$LECTURE_ID" "$OUTDOOR_ID" "$KITCHEN_ID"; do
    echo "DELETE /maps/$id"
    curl -sS -A "$UA" -H 'Accept: application/json' -X DELETE "$BASE/maps/$id" -i | head -n1
  done
else
  say "12) CLEANUP=0 のため削除をスキップしました"
fi

say "✅ 完了"
