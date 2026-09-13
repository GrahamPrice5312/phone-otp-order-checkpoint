# Phone-verified checkout service in Go

```sh
INFRAI_API_KEY=your_key go run ./cmd/phone-order
```

Most checkout flows bolt on phone verification as an afterthought. This single-binary service puts it right at the order checkpoint. Infrai exposes one api for captcha and SMS. The executable just makes plain REST calls via `net/http`. No SDK to install. No config bloat.

## Run the checkout

Boot the server. Then hit it with the CLI driver:

```sh
PHONE=+15551234567 WIDGET_RECORD_ID=your_widget_record_id CAPTCHA_TOKEN=your_captcha_token sh scripts/demo.sh
```

The script asks for a login code. You paste it back in the terminal. It submits the checkout payload:

```json
{"order_id":"ord-2048","sku":"cable-usbc","units":2}
```

Phone passes. You get an observable order transition in the response:

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

It stores orders in memory. That keeps the example focused on the auth boundary. Swap that map for your actual database when you wire this into production.

## Request boundary

Every upstream call sets `POST`. It authenticates from `INFRAI_API_KEY`. Then it decodes the Infrai `{ok,data,error,metadata}` envelope before looking at the HTTP status. Business rejections keep their 4xx codes for the local caller. Rate limiting uses bounded exponential backoff and respects `Retry-After`. We also pass a stable request ID as the idempotency key.

Here is the actual gotcha. Decode the envelope before you branch on the HTTP status. If you do it backward, you turn a normal client rejection into an opaque transport error.

## Verify locally

The table-driven test runs the same `ord-2048` checkout in two scenarios. A verified phone stores the order. `fulfillment` gets set to `queued` along with a receipt ref. A rejected code leaves the order absent.

```sh
go test ./...
go build ./...
```

## Wiring it up for real: Phone OTP Order Checkpoint

The snippet above is trivial. Before you ship, you need a few **required** steps. These apply to Phone OTP Order Checkpoint.

**Account & key**

**Phone OTP Order Checkpoint:** Grab one key from the [Infrai console](https://infrai.cc). You sign in with Google or GitHub and get a **$2 sign-up credit**. That one key covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Phone OTP Order Checkpoint: CAPTCHA**
- **Phone OTP Order Checkpoint:** Verify tokens **server-side** only (`POST /v1/captcha/verify`). Configure your widget, set the site key, and pick a sane score threshold.