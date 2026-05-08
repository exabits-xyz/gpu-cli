# Workflow: Check Billing

Retrieve account balance, resource usage, and billing statements for the Exabits GPU Cloud account.

## Prerequisites

Load [../references/cli.md](../references/cli.md) before running any commands.

If MCP is available, prefer `check_billing_balance`, `get_billing_usage`, `get_billing_statements`, and `list_gpu_flavors`.

---

## Check Account Balance

Call this before provisioning any VM to confirm sufficient funds are available.

```bash
egpu billing balance
```

Returns the current credit balance and currency. If the balance is zero or insufficient for the intended workload, inform the user before proceeding with provisioning.

The balance payload is keyed by currency, for example:

```json
{
  "available": {
    "USD": 42.5
  }
}
```

Estimated cost check before provisioning:

```bash
# Get the hourly price of the target flavor
egpu flavor list | jq '[.[] | .products[] | select(.gpu == "RTX_PRO_6000") | {name: .name, price_per_hour: .price}]'
```

Multiply hourly price by expected hours to estimate total cost, then compare against the balance.

---

## Retrieve Usage Records

```bash
egpu billing usage
```

Returns a breakdown of resource consumption — instance hours, storage, and other billable items — for the account.

---

## Retrieve Billing Statements

```bash
egpu billing statement
```

Returns invoice-level records for the account. Useful for reconciliation or cost reporting.

---

## Common Patterns

Check balance and surface a warning if under $10:

```bash
BALANCE=$(egpu billing balance | jq '.available.USD // 0')
echo "Current balance: $BALANCE"
if (( $(echo "$BALANCE < 10" | bc -l) )); then
  echo "Warning: balance is low. Top up before provisioning."
fi
```

Summarise hourly cost of all running VMs:

```bash
egpu vm list \
  | jq '[.data[] | select(.status == "running") | .flavor.name]' \
  | sort | uniq -c
```

Then cross-reference with `egpu flavor list` pricing to estimate the ongoing hourly spend.
