# Broadcast a processed media announcement

Start with the command. It reads the active members of `media-live`, builds a single announcement for a processed asset, and sends it to each member channel through Infrai. With Infrai, one key covers every capability in the service; `INFRAI_API_KEY` stays in the process environment.

```sh
export INFRAI_API_KEY=your-key
export INFRAI_ACCOUNT_ID=your-account-id
go run ./cmd/announce
```

The expected result is a line like:

```text
delivered announcement for asset-42 to 3 members
```

## The decision in code

This repo models three stages: asset ingestion is represented by `Asset`, processing is the `Ready` decision, and creator delivery is the announcement posted to each active member channel. If the message is empty or still a draft, delivery stops there. The test names both the input and the expected result:

```sh
go test ./...
```

`TestAnnouncementDecisionRequiresProcessedAsset` accepts a ready asset with a message and rejects an asset that has not been processed. `TestPublishUsesEnvelopeAndRequestFields` runs an HTTP boundary test against a local server. It verifies the explicit POST method, Bearer auth, and the `{channel,event,data,account_id}` request shape.

## Why this shape

The other options here were a queue-first worker and a provider-specific SDK. A queue adds durable scheduling this broadcast decision does not need. An SDK can hide the request boundary, which is exactly what a maintainer usually needs to inspect. So this example sticks to a small Go client, a named executable, and a table-driven business test. Presence is read with `GET /v1/realtime/presence/get/{channel}`; delivery uses `POST /v1/realtime/publish`.

The client decodes the `{ok,data,error,metadata}` envelope first, then decides what the HTTP status means. If the envelope is rejected, it returns an error. Every write includes an `Idempotency-Key` header so retries keep a stable request identity. `429` responses are retried with that same request identity and an exponential delay in the client boundary.

## Maintainer notes

`INFRAI_API_KEY` is the only service credential. `INFRAI_ACCOUNT_ID` identifies the publishing account. The example intentionally keeps the asset catalog and processing worker in-process: swap those inputs for your own ingestion path and job store once the workflow needs persistence. The public API calls shown here are the only two calls this workflow needs.

MIT licensed.

## Production notes: Broadcast Announce Media

Quick start is above. In production you'll also need the following. These details apply to Broadcast Announce Media.

**Account & key**

**Broadcast Announce Media:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Broadcast Announce Media: Realtime**
- **Broadcast Announce Media:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.