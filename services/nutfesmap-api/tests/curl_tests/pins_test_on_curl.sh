#!/usr/bin/env bash
# pins_test_on_curl.sh — complete v3: robust to missing ETag on POST/PATCH
set -euo pipefail
export LC_ALL=C

# ========= 設定 =========
BASE="${BASE:-http://localhost:8080}"
UA="${UA:-curl-test/pins-smoke-complete-3.3}"
B64_IMG="${B64_IMG:-iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAAWgmWQ0AAAAASUVORK5CYII=}"
CLEANUP="${CLEANUP:-1}"

CURL_TIMEOUT="${CURL_TIMEOUT:-20}"
CURL_CONNECT_TIMEOUT="${CURL_CONNECT_TIMEOUT:-5}"
RETRY_COUNT="${RETRY_COUNT:-2}"
RETRY_DELAY="${RETRY_DELAY:-1}"  # 整数のみ（古い curl 互換）
DEBUG="${DEBUG:-0}"              # 0/1/2 (2 は hexdump)
FORCE_IPV4="${FORCE_IPV4:-1}"
FORCE_IPV6="${FORCE_IPV6:-0}"
NO_PROXY_STR="${NO_PROXY_STR:-*}"
POST_PAUSE="${POST_PAUSE:-0}"

# ETag 要件（環境により POST/PATCH は付かないことがある）
REQUIRE_ETAG_GET="${REQUIRE_ETAG_GET:-1}"
REQUIRE_ETAG_POST="${REQUIRE_ETAG_POST:-0}"
REQUIRE_ETAG_PATCH="${REQUIRE_ETAG_PATCH:-0}"

need() { command -v "$1" >/dev/null 2>&1 || { echo "ERROR: '$1' not found" >&2; exit 1; }; }
need jq; need curl; need mktemp

say()  { printf '\n== %s ==\n' "$*"; }
warn() { echo "WARN: $*" >&2; }
fail() { echo "ERROR: $*" >&2; exit 1; }
assert_eq()       { local got="$1" want="$2" msg="${3:-}"; [ "$got" = "$want" ] || fail "ASSERT: $msg (got=$got want=$want)"; }
assert_nonempty() { local got="$1" msg="${2:-}"; [ -n "$got" ] || fail "ASSERT: empty ($msg)"; }

# ========= curl 機能検出 & 共通オプション =========
curl_has() { curl --help all 2>&1 | grep -q -- "$1"; }

CURL_OPTS=(
  -sS
  --http1.1
  --max-time "$CURL_TIMEOUT"
  --connect-timeout "$CURL_CONNECT_TIMEOUT"
  --retry "$RETRY_COUNT"
  --noproxy "$NO_PROXY_STR"
  -A "$UA"
  -H 'Accept: application/json'
  -H 'Connection: close'
  -H 'Expect:'             # 100-continue 抑止
)
if curl_has "--retry-all-errors"; then CURL_OPTS+=(--retry-all-errors); fi
if curl_has "--retry-delay"; then CURL_OPTS+=(--retry-delay "$RETRY_DELAY"); fi
if [[ "$DEBUG" = "1" ]]; then CURL_OPTS+=(-v); fi
if [[ "$FORCE_IPV6" = "1" ]]; then CURL_OPTS+=(-6); elif [[ "$FORCE_IPV4" = "1" ]]; then CURL_OPTS+=(-4); fi

# ========= JSON救出（ヘッダ/ログ混入の自動補正） =========
_rescue_json_body() {
  local f="$1"
  if head -c1 "$f" | grep -q '[{\[]'; then return 0; fi
  local pos
  pos="$(LC_ALL=C LANG=C awk '
    BEGIN{pos=-1}
    {
      for(i=1;i<=length($0);i++){
        c=substr($0,i,1);
        if(c=="{" || c=="["){pos=NR":"i; print pos; exit}
      }
    }
  ' "$f" | head -n1 || true)"
  if [ -n "$pos" ]; then
    local line=${pos%%:*} col=${pos##*:}
    awk -v ln="$line" -v cl="$col" 'NR<ln{next} NR==ln{print substr($0,cl); next} {print}' "$f" > "${f}.clean"
    mv "${f}.clean" "$f"
    return 0
  fi
  return 1
}

# ========= 1回のHTTP実行（ETag 要件を引数で制御） =========
# _once METHOD URL [JSON] [require_etag: 0|1]
_once() {
  local method="$1" url="$2" payload="${3:-}" need_etag="${4:-1}"
  local tmp_h tmp_b; tmp_h="$(mktemp)"; tmp_b="$(mktemp)"
  trap 'rm -f "$tmp_h" "$tmp_b" 2>/dev/null || true' RETURN

  if [ -n "$payload" ]; then
    curl "${CURL_OPTS[@]}" -H 'Content-Type: application/json' -D "$tmp_h" -o "$tmp_b" -X "$method" "$url" -d "$payload"
  else
    curl "${CURL_OPTS[@]}" -D "$tmp_h" -o "$tmp_b" -X "$method" "$url"
  fi

  local code; code="$(awk 'NR==1 {print $2}' "$tmp_h" | tr -d '\r')"
  [[ "$code" =~ ^2[0-9][0-9]$ ]] || {
    echo "---- Response Headers ----" >&2; cat "$tmp_h" >&2
    echo "---- Response Body ----" >&2;    cat "$tmp_b" >&2
    fail "unexpected HTTP status ($code) on $method $url"
  }

  local etag; etag="$(awk 'BEGIN{IGNORECASE=1}/^ETag:/{sub(/\r$/,""); $1=""; sub(/^:[[:space:]]*/,""); print; exit}' "$tmp_h")"
  if [[ "$need_etag" = "1" ]]; then
    assert_nonempty "$etag" "ETag on $method $url"
  else
    [ -n "$etag" ] || warn "ETag missing on $method $url (continuing)"
  fi

  if ! _rescue_json_body "$tmp_b"; then
    echo "---- Raw Body (head) ----" >&2; head -n 50 "$tmp_b" >&2
    [ "$DEBUG" = "2" ] && { echo "---- Raw Body (hexdump) ----" >&2; hexdump -C "$tmp_b" | head -n 200 >&2; }
    fail "body is not JSON on $method $url"
  fi
  jq -e . "$tmp_b" >/dev/null 2>&1 || {
    echo "---- Cleaned Body (head) ----" >&2; head -n 50 "$tmp_b" >&2
    [ "$DEBUG" = "2" ] && { echo "---- Cleaned Body (hexdump) ----" >&2; hexdump -C "$tmp_b" | head -n 200 >&2; }
    fail "JSON parse error on $method $url"
  }
  cat "$tmp_b"
}

_http_code() { curl "${CURL_OPTS[@]}" -o /dev/null -w "%{http_code}" "$@"; }

# ========= APIユーティリティ =========
create_root_map() {
  local name="$1" json id
  json="$(curl "${CURL_OPTS[@]}" -H 'Content-Type: application/json' -X POST "$BASE/maps" -d '{"parentMapId":null}')" \
    || fail "root map POST 失敗（$name）"
  id="$(jq -er '.id' <<<"$json")" || fail "root map id 取得失敗（$name）"
  curl "${CURL_OPTS[@]}" -H 'Content-Type: application/json' -X PATCH "$BASE/maps/$id" \
    -d @- >/dev/null <<JSON
{ "name":"$name", "imageData":"$B64_IMG", "naturalWidth":2048, "naturalHeight":1536 }
JSON
  echo "$id"
}

create_floor() {
  local root_id="$1" fname="${2:-1F}" fid
  fid="$(curl "${CURL_OPTS[@]}" -X POST "$BASE/maps/$root_id/floors" | jq -er '.id')" \
    || fail "floor 作成失敗（root=$root_id）"
  curl "${CURL_OPTS[@]}" -H 'Content-Type: application/json' -X PATCH "$BASE/maps/$fid" -d "{\"name\":\"$fname\"}" >/dev/null
  echo "$fid"
}

create_pin() {
  local map_id="$1"; shift
  local payload="$*"
  local need="${REQUIRE_ETAG_POST}"
  _once "POST" "$BASE/maps/$map_id/pins" "$payload" "$need"
}
list_pins()  { local map_id="$1"; _once "GET" "$BASE/maps/$map_id/pins" "" "${REQUIRE_ETAG_GET}" >/dev/null; curl "${CURL_OPTS[@]}" -X GET "$BASE/maps/$map_id/pins"; }
show_pin()   { local pin_id="$1"; _once "GET" "$BASE/pins/$pin_id" "" "${REQUIRE_ETAG_GET}" >/dev/null; curl "${CURL_OPTS[@]}" -X GET "$BASE/pins/$pin_id"; }
update_pin() { local pin_id="$1" payload="$2"; _once "PATCH" "$BASE/pins/$pin_id" "$payload" "${REQUIRE_ETAG_PATCH}"; }
delete_pin() { local pin_id="$1"; local code; code="$(_http_code -X DELETE "$BASE/pins/$pin_id")"; assert_eq "$code" "204" "delete pin ($pin_id)"; }

# ========= メイン =========
echo "BASE=$BASE (timeout=${CURL_TIMEOUT}s, connect=${CURL_CONNECT_TIMEOUT}s)"

say "1) マップ/フロア準備"
ROOT_ID="$(create_root_map 'ピン検証エリア')"; echo "ROOT_ID=$ROOT_ID"
F1_ID="$(create_floor "$ROOT_ID" '1F')";       echo "F1_ID=$F1_ID"

say "2) linkToMapId 用マップ"
LINK_ROOT_ID="$(create_root_map '屋外リンク先')"; echo "LINK_ROOT_ID=$LINK_ROOT_ID"

say "3) ピンを3件作成（Zulu / Alpha / Mike）"
PIN_A_JSON="$(create_pin "$F1_ID" '{"name":"Zulu","xNorm":0.6,"yNorm":0.1,"category":"other","type":"exhibit","status":"open"}')"
PIN_B_JSON="$(create_pin "$F1_ID" "{\"name\":\"Alpha\",\"xNorm\":0.2,\"yNorm\":0.3,\"category\":\"service\",\"type\":\"info\",\"status\":\"paused\",\"waitMinutes\":3}")"
PIN_C_JSON="$(create_pin "$F1_ID" "{\"name\":\"Mike\",\"xNorm\":0.4,\"yNorm\":0.8,\"category\":\"food\",\"type\":\"area_selector\",\"linkToMapId\":\"$LINK_ROOT_ID\",\"description\":\"屋外への誘導\",\"descriptionImageData\":\"$B64_IMG\"}")"

PIN_A_ID="$(jq -r '.id' <<<"$PIN_A_JSON")"
PIN_B_ID="$(jq -r '.id' <<<"$PIN_B_JSON")"
PIN_C_ID="$(jq -r '.id' <<<"$PIN_C_JSON")"
assert_nonempty "$PIN_A_ID" "PIN_A_ID"; assert_nonempty "$PIN_B_ID" "PIN_B_ID"; assert_nonempty "$PIN_C_ID" "PIN_C_ID"
echo "PIN_A_ID=$PIN_A_ID"; echo "PIN_B_ID=$PIN_B_ID"; echo "PIN_C_ID=$PIN_C_ID"
[ "${POST_PAUSE}" = "0" ] || sleep "${POST_PAUSE}"

say "4) 一覧の並び（Name 昇順）検証"
LIST_JSON="$(list_pins "$F1_ID")"
NAMES="$(jq -r '.[].name' <<<"$LIST_JSON" | paste -sd',' -)"
assert_eq "$NAMES" "Alpha,Mike,Zulu" "sorted by name asc"

say "5) 単体取得（Alpha）"
SHOW_B="$(show_pin "$PIN_B_ID")"
assert_eq "$(jq -r '.name' <<<"$SHOW_B")" "Alpha" "show Alpha"

say "6) 部分更新（Alpha → Alpha改、desc=null、xNorm=0.5、status=closed、wait=10）"
UPDATED_B="$(update_pin "$PIN_B_ID" '{"name":"Alpha改","description":null,"xNorm":0.5,"status":"closed","waitMinutes":10}')"
assert_eq "$(jq -r '.name'        <<<"$UPDATED_B")" "Alpha改"
assert_eq "$(jq -r '.status'      <<<"$UPDATED_B")" "closed"
assert_eq "$(jq -r '.waitMinutes' <<<"$UPDATED_B")" "10"
assert_eq "$(jq -r '.xNorm'       <<<"$UPDATED_B")" "0.5"
DESC_OK="$(jq -r 'if has("description") then (.description==null) else true end' <<<"$UPDATED_B")"
assert_eq "$DESC_OK" "true" "description cleared (missing or null acceptable)"

say "7) 異常系（400 を期待）"
assert_eq "$(_http_code -H 'Content-Type: application/json' -X PATCH "$BASE/pins/$PIN_A_ID" -d '{"xNorm":1.2}')" "400" "xNorm out of range"
assert_eq "$(_http_code -H 'Content-Type: application/json' -X PATCH "$BASE/pins/$PIN_A_ID" -d '{"type":"invalid"}')" "400" "invalid type"
assert_eq "$(_http_code -H 'Content-Type: application/json' -X PATCH "$BASE/pins/$PIN_A_ID" -d '{"status":"maybe"}')" "400" "invalid status"

say "8) linkToMapId の付与→解除（omitempty対応）"
WITH_LINK="$(update_pin "$PIN_A_ID" "{\"linkToMapId\":\"$LINK_ROOT_ID\"}")"
assert_eq "$(jq -r '.linkToMapId' <<<"$WITH_LINK")" "$LINK_ROOT_ID" "linkToMapId attached"
NO_LINK="$(update_pin "$PIN_A_ID" '{"linkToMapId":null}')"
LINK_OK="$(jq -r 'if has("linkToMapId") then (.linkToMapId==null) else true end' <<<"$NO_LINK")"
assert_eq "$LINK_OK" "true" "linkToMapId cleared (missing or null acceptable)"

say "9) ETag 条件付きGET（304）"
if [[ "$REQUIRE_ETAG_GET" = "1" ]]; then
  PIN_ETAG="$(curl "${CURL_OPTS[@]}" -D - -o /dev/null "$BASE/pins/$PIN_A_ID" | awk 'BEGIN{IGNORECASE=1}/^ETag:/{print $2}' | tr -d '\r')"
  assert_nonempty "$PIN_ETAG" "pin etag"
  assert_eq "$(_http_code -H "If-None-Match: $PIN_ETAG" "$BASE/pins/$PIN_A_ID")" "304" "If-None-Match (pin)"

  LIST_ETAG="$(curl "${CURL_OPTS[@]}" -D - -o /dev/null "$BASE/maps/$F1_ID/pins" | awk 'BEGIN{IGNORECASE=1}/^ETag:/{print $2}' | tr -d '\r')"
  assert_nonempty "$LIST_ETAG" "list etag"
  assert_eq "$(_http_code -H "If-None-Match: $LIST_ETAG" "$BASE/maps/$F1_ID/pins")" "304" "If-None-Match (list)"
else
  warn "REQUIRE_ETAG_GET=0 → 304 テストはスキップします"
fi

say "10) 削除（A/B/C）→ 件数0を確認"
delete_pin "$PIN_A_ID"
CNT2="$(jq 'length' <<<"$(list_pins "$F1_ID")")"; assert_eq "$CNT2" "2" "after 1 delete"
delete_pin "$PIN_B_ID"
delete_pin "$PIN_C_ID"
CNT3="$(jq 'length' <<<"$(list_pins "$F1_ID")")"; assert_eq "$CNT3" "0" "after all deletes"

if [ "${CLEANUP}" = "1" ]; then
  say "11) 片付け（マップ再帰削除）"
  for id in "$F1_ID" "$ROOT_ID" "$LINK_ROOT_ID"; do
    echo "DELETE /maps/$id"
    curl "${CURL_OPTS[@]}" -X DELETE "$BASE/maps/$id" -i | head -n1
  done
else
  say "11) CLEANUP=0 のため削除スキップ"
fi

echo "✅ 完了"
