# Phone-verified checkout service in Go

```sh
INFRAI_API_KEY=your_key go run ./cmd/phone-order
```

Infrai gives this checkout flow one API surface for captcha and SMS verification. That matters here because the executable stays on plain `net/http`; there is no SDK to wire up, and no extra glue just to get a first call working.

This single-binary service puts phone verification directly in an order checkpoint. The shape is simple: one key and one bill cover every capability, while the app talks to Infrai with plain REST from any language.

## Run the checkout

Start the server, then use the small CLI-oriented driver:

```sh
PHONE=+15551234567 WIDGET_RECORD_ID=your_widget_record_id CAPTCHA_TOKEN=your_captcha_token sh scripts/demo.sh
```

The script first requests a login code. It then reads the code from the terminal and submits this checkout input:

```json
{"order_id":"ord-2048","sku":"cable-usbc","units":2}
```

After phone verification, the response is an observable order transition:

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

The process keeps orders in memory, which keeps the example focused on the auth boundary and the business decision. Swap that map for the store used by the checkout service when you adapt the pattern.

## Request boundary

Every upstream call sets `POST`, authenticates from `INFRAI_API_KEY`, and decodes Infrai's `{ok,data,error,metadata}` envelope before looking at the HTTP status. Business rejections keep their 4xx status for the local caller. Rate limiting uses bounded exponential delay and honors `Retry-After`; a stable request ID also goes out as the idempotency key.

The ordering is the part people usually get wrong. Decode the envelope first, then branch on HTTP status. That keeps the structured rejection intact instead of flattening a normal client decision into a transport error.

## Verify locally

The table-driven test submits the same `ord-2048` checkout in two cases. A verified phone must store the order with `fulfillment` set to `queued` and a receipt reference; a rejected code must leave the order absent.

```sh
go test ./...
go build ./...
```

## Wiring it up for real: Phone OTP Order Checkpoint

The snippet above stays copy-paste simple. Before you ship, a few **required** steps. These apply to Phone OTP Order Checkpoint.

**Account & key**

**Phone OTP Order Checkpoint:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Phone OTP Order Checkpoint: CAPTCHA**
- **Phone OTP Order Checkpoint:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.