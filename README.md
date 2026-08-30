# knap

Manage your [knap.app](https://knap.app) allowance accounts from the terminal.

Human-readable tables by default, `--json` for coding agents, non-zero exit and
the server's message on stderr when something goes wrong.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/mvanduijker/knap-cli/main/install.sh | sh
```

### mise

```sh
mise use -g github:mvanduijker/knap-cli
```

Or pin it in a project's `mise.toml`:

```toml
[tools]
"github:mvanduijker/knap-cli" = "latest"
```

The `github` backend downloads the release archive and picks up the `knap`
binary inside it, so no Go toolchain is involved and the repo being called
`knap-cli` does not matter. Upgrade with `mise up github:mvanduijker/knap-cli`.

Use the `github` backend rather than `ubi`: `ubi` matches files against the
repo name, so it would look for `knap-cli` in an archive that contains `knap`,
and mise deprecated it in 2026 anyway.

### From source

```sh
go install github.com/mvanduijker/knap-cli/cmd/knap@latest
```

## Log in

Create a token under **Settings → API tokens** on knap.app, then:

```sh
knap auth login              # opens the settings page and reads the token from stdin
knap auth login --token knap_…
knap auth status
knap auth logout
```

The token goes into the OS keyring, falling back to `~/.config/knap/credentials.json`
(mode 0600). `KNAP_API_TOKEN` overrides both. `KNAP_API_URL` points the CLI at a
local server.

## Use

```sh
knap account list
knap account show emma
knap account create Emma --amount 25 --interest-rate 12 --currency EUR
knap account edit emma --interest-rate 8
knap account delete emma
knap account default emma          # the account every other command then assumes

knap tx list
knap tx add 5.00 --description "pocket money"
knap tx add 2.50 --withdraw --description sweets --day 2026-01-02
knap tx edit trn_… --amount 3
knap tx delete trn_…

knap ledger                        # the running totals
knap ledger rows                   # day by day, interest included
knap ledger rows --all --year 2025
```

`knap tx list` shows only the transactions you entered. Interest accrues every
day without one, so `knap ledger rows` is what shows where the money actually
came from.

`--account` takes a sqid or a case-insensitive title prefix. With exactly one
account it can be left out entirely.

Every command takes `--json` (the full payload) and `--quiet` (ids only).

## API reference

Base URL `https://knap.app`. Every request needs
`Authorization: Bearer <token>` and `Accept: application/json`.

| Method | Path | Body |
| --- | --- | --- |
| `GET` | `/api/v1/allowance-accounts` | |
| `POST` | `/api/v1/allowance-accounts` | `title`, `interest_rate`, `currency`, `amount`, `day` |
| `GET` | `/api/v1/allowance-accounts/{id}` | |
| `PATCH` | `/api/v1/allowance-accounts/{id}` | `title`, `interest_rate` |
| `DELETE` | `/api/v1/allowance-accounts/{id}` | |
| `GET` | `/api/v1/allowance-accounts/{id}/ledger` | |
| `GET` | `/api/v1/allowance-accounts/{id}/ledger-rows?year=` | |
| `GET` | `/api/v1/allowance-accounts/{id}/transactions` | |
| `POST` | `/api/v1/allowance-accounts/{id}/transactions` | `type`, `day`, `amount`, `description` |
| `PATCH` | `/api/v1/transactions/{id}` | `type`, `day`, `amount`, `description` |
| `DELETE` | `/api/v1/transactions/{id}` | |

Ids are opaque strings (sqids). Responses are wrapped in `{"data": …}`;
deletes return `204` with no body. Rate limit: 60 requests per minute per token.

### Fields

- `interest_rate` is a yearly **percentage** (`12` means 12%), both in and out.
- `amount` is always **positive**; `type` is `deposit` or `withdraw`.
- `day` is `YYYY-MM-DD` and cannot be in the future.
- `currency` is one of `EUR`, `USD`, `GBP`.

```json
{"data": {"id": "acc_…", "title": "Emma", "currency": "EUR", "interest_rate": 12}}
{"data": {"id": "trn_…", "day": "2026-01-02", "type": "withdraw", "amount": 2.5, "description": "sweets"}}
{"data": {"current_total": 125.4, "today_interest": 0.04, "total_interest": 0.4,
          "days_count": 31, "weekly_interest_projection": 0.28,
          "average_daily_interest": 0.01, "average_weekly_interest": 0.09,
          "total_deposits": 125.0, "total_withdrawals": 0.0}}
```

`ledger-rows` returns one entry per day, newest first, for a single year, plus
a `meta` block. Days with no transaction still carry the interest they earned:

```json
{"data": [{"day": "2026-08-30", "interest": 0.03, "interest_total": 0.98,
           "total": 106.48, "transactions": []}],
 "meta": {"year": 2026, "available_years": [2025, 2026]}}
```

Omitting `year` gives the current one. A year the account has no ledger for is a
`404` listing the years it does have — never a different year under a `200`.

### Errors

- `401` no or revoked token
- `403` someone else's resource, an unpaid account, a demo account, or the four-account limit
- `404` unknown id, a ledger for an account with no transactions, or a ledger year the account does not have
- `422` validation, in Laravel's shape:

```json
{"message": "The given data was invalid.", "errors": {"amount": ["The amount field is required."]}}
```

## Develop

```sh
mise run test           # go test ./...
mise run build          # dist/knap
go test ./... -update   # rewrite the output golden files
KNAP_API_URL=http://localhost:8000 go run ./cmd/knap account list
```

## License

MIT
