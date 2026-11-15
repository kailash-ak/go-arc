#!/usr/bin/env bash
# URL shortener API smoke tests
# Start the Go server in another terminal with: go run .
# Run this script with: bash test_urls.sh

set -u
BASE_URL="${BASE_URL:-http://localhost:8080}"
TMP_BODY="$(mktemp)"
trap 'rm -f "$TMP_BODY"' EXIT

section() {
  printf '\n\n============================================================\n%s\n============================================================\n' "$1"
}

request() {
  printf '\n$'
  printf ' %q' "$@"
  printf '\n'
  curl --silent --show-error --include "$@" || true
}

section "0. Check server (root may return 404 if you have no root route)"
request "$BASE_URL/"

section "1. Create URL (expected 201 Created and Location header)"
CREATE_RESPONSE="$(curl --silent --show-error --include \
  -X POST "$BASE_URL/urls" \
  -H 'Content-Type: application/json' \
  --data-raw '{"longUrl":"https://example.com/hello"}')"
printf '%s\n' "$CREATE_RESPONSE"

CREATE_BODY="$(printf '%s\n' "$CREATE_RESPONSE" | sed '1,/^\r\{0,1\}$/d')"
CODE="$(printf '%s' "$CREATE_BODY" | python3 -c \
  'import json,sys; print(json.load(sys.stdin)["code"])' 2>/dev/null || true)"

if [[ -n "$CODE" ]]; then
  echo "Extracted short code: $CODE"
  section "2. Get existing URL (expected 200 OK)"
  request "$BASE_URL/urls/$CODE"

  section "3. Get nonexistent code (expected 404 Not Found)"
  request "$BASE_URL/urls/doesNotExist"
else
  echo "Could not extract a short code; check the POST response."
fi

section "4. List URLs (expected 200 OK and a JSON array)"
request "$BASE_URL/urls"

section "5. application/json with charset (expected 201 Created)"
request -X POST "$BASE_URL/urls" \
  -H 'Content-Type: application/json; charset=utf-8' \
  --data-raw '{"longUrl":"https://example.org/charset-test"}'

section "6. Wrong Content-Type text/plain (expected 415)"
request -X POST "$BASE_URL/urls" \
  -H 'Content-Type: text/plain' \
  --data-raw '{"longUrl":"https://example.com"}'

section "7. Missing Content-Type (expected 415)"
request -X POST "$BASE_URL/urls" \
  -H 'Content-Type:' \
  --data-raw '{"longUrl":"https://example.com"}'

section "8. Malformed JSON (expected 400)"
request -X POST "$BASE_URL/urls" \
  -H 'Content-Type: application/json' \
  --data-raw '{"longUrl":'

section "9. Invalid URL scheme (expected 400)"
request -X POST "$BASE_URL/urls" \
  -H 'Content-Type: application/json' \
  --data-raw '{"longUrl":"ftp://example.com/file"}'

section "10. Empty URL (expected 400)"
request -X POST "$BASE_URL/urls" \
  -H 'Content-Type: application/json' \
  --data-raw '{"longUrl":"   "}'

section "11. Oversized body (expected 413)"
python3 -c 'import json; print(json.dumps({"longUrl":"https://example.com","extra":"A"*1100000}))' > "$TMP_BODY"
request -X POST "$BASE_URL/urls" \
  -H 'Content-Type: application/json' \
  --data-binary "@$TMP_BODY"

if [[ -n "$CODE" ]]; then
  section "12. Delete existing URL (expected 204 No Content)"
  request -X DELETE "$BASE_URL/urls/$CODE"

  section "13. Get deleted URL (expected 404 Not Found)"
  request "$BASE_URL/urls/$CODE"

  section "14. Delete same URL again (expected 404 Not Found)"
  request -X DELETE "$BASE_URL/urls/$CODE"
fi

section "15. Final list (expected 200 OK)"
request "$BASE_URL/urls"

echo
echo "Smoke tests finished. Compare the HTTP status lines with the expected statuses."
