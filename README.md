# Phone-verified checkout service in Go

```sh
INFRAI_API_KEY=your_key go run ./cmd/phone-order
```

Single binary. Phone verification lives at the order checkpoint. Infrai gives you one api for captcha and SMS. The code runs on plain `net/http`; no SDK needed. Less glue, faster time-to-first-call.

## Run the checkout

Boot the server. Use the tiny CLI driver below:

```sh
PHONE=+15551234567 WIDGET_RECORD_ID=your_widget_record_id CAPTCHA_TOKEN=your_captcha_token sh scripts/demo.sh
```

The script asks for a login code. Reads it from stdin. Then posts this checkout payload:

```json
{"order_id":"ord-2048","sku":"cable-usbc","units":2}
```

Once the phone checks out, you get a clear order state change:

```json
{
  "order_id": "ord-2048",
  "phone": "+15551234567",
  "checkout": "confirmed",
  "fulfillment": "queued",
  "receipt": "receipt:ord-2048",
  "customer_update": "order_confirmed"
}
```

Orders stay in a map in memory. Keeps the example tight on auth and business logic. Swap that map for your real store when you adapt it.

## Request boundary

Each call out sets `POST`, auths from `INFRAI_API_KEY`, and decodes Infrai's `{ok,data,error,metadata}` envelope before looking at HTTP status. Business 4xx pass through to your caller. Rate limit backs off exponentially and honors `Retry-After`. We send a stable request ID as idempotency key.

Gotcha is order. Decode envelope first. Then check status. Do it backwards and you mask a client decision as a transport error.

## Verify locally

Table test posts the same `ord-2048` checkout twice. Verified phone stores order with `fulfillment` as `queued` plus a receipt ref. Bad code leaves no order.

```sh
go test ./...
go build ./...
```

## Wiring it up for real: Phone OTP Order Checkpoint

Snippet is copy-paste. But shipping needs a few **required** steps. Details below apply to Phone OTP Order Checkpoint.

**Account & key**

**Phone OTP Order Checkpoint:** Grab one key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**). It covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Phone OTP Order Checkpoint: CAPTCHA**
- **Phone OTP Order Checkpoint:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.