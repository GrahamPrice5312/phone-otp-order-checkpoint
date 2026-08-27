#!/bin/sh
set -eu

: "${PHONE:?set PHONE}"
: "${WIDGET_RECORD_ID:?set WIDGET_RECORD_ID}"
: "${CAPTCHA_TOKEN:?set CAPTCHA_TOKEN}"

curl --fail-with-body -X POST http://localhost:8080/login/code \
  -H 'Content-Type: application/json' \
  -d "{\"phone\":\"$PHONE\",\"widget_record_id\":\"$WIDGET_RECORD_ID\",\"captcha_token\":\"$CAPTCHA_TOKEN\",\"request_id\":\"demo-send-1\"}"

printf 'Code: '
read -r OTP_CODE

curl --fail-with-body -X POST http://localhost:8080/checkout/confirm \
  -H 'Content-Type: application/json' \
  -d "{\"phone\":\"$PHONE\",\"code\":\"$OTP_CODE\",\"request_id\":\"demo-verify-1\",\"checkout\":{\"order_id\":\"ord-2048\",\"sku\":\"cable-usbc\",\"units\":2}}"
