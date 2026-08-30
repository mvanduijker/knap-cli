---
name: knap
description: Read and change knap.app allowance accounts, transactions and ledgers through the `knap` CLI. Use when the user asks about their kids' allowance, pocket money, savings balances or interest on knap.app, or asks to record a deposit or withdrawal. Requires the `knap` binary and a stored API token.
---

# knap

`knap` is a CLI over the knap.app JSON API. Always pass `--json` — every command
then prints the full payload and nothing else on stdout.

Check it is usable before anything else:

```sh
knap auth status --json
```

A non-zero exit means either the binary is missing (install it from
https://github.com/mvanduijker/knap-cli) or there is no token — tell the user to
create one under Settings → API tokens on knap.app and run
`knap auth login --token knap_…`. Never ask them to paste a token into the chat.

## Reading

```sh
knap account list --json
knap ledger emma --json              # running totals
knap ledger rows emma --json         # day by day, interest included
knap tx list emma --json             # only the entered transactions
```

`tx list` does **not** explain a balance on its own — interest accrues every day
with no transaction behind it. To account for where money came from, use
`ledger rows`, which returns one entry per day for a single year (newest first,
`--all` for the whole year, `--year YYYY` for an earlier one). Asking for a year
the account has no ledger for exits non-zero and lists the years it does have —
read those from `meta.available_years` rather than guessing.

`account` takes a sqid (`acc_…`) or a case-insensitive title prefix. With one
account it can be left out.

## Writing

Changes touch real money, so confirm the amount, the direction and the account
with the user before running any of these.

```sh
knap tx add 5.00 --account emma --description "pocket money" --json
knap tx add 2.50 --account emma --withdraw --description "sweets" --json
knap tx edit trn_… --account emma --amount 3.00 --json
knap tx delete trn_… --json
```

`--day YYYY-MM-DD` backdates; it cannot be in the future. Amounts are always
positive — `--withdraw` sets the direction.

## Shapes

- `interest_rate` is a yearly percentage (`12` means 12%).
- Transactions: `{id, day, type: deposit|withdraw, amount, description}`.
- Ledger rows: `{day, interest, interest_total, total, transactions[]}` — the
  interest a day earned, the cumulative interest, and the running balance.
- Ledger: `{current_total, today_interest, total_interest, days_count,
  weekly_interest_projection, average_daily_interest, average_weekly_interest,
  total_deposits, total_withdrawals}`.

## Errors

Failures exit non-zero and print the server's message on stderr. Relay it as it
is instead of retrying: `403` means an unpaid account, someone else's resource,
or the four-account limit; `422` lists the invalid fields; `429` means the
60-per-minute rate limit — wait, do not loop.
