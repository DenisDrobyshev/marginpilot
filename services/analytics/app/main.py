"""Margin Analytics service.

The Python home of the platform's analytics/ML work:
  - per-customer gross margin: plan revenue (PostgreSQL) vs AI COGS (ClickHouse)
  - spend-anomaly detection: rolling mean + standard deviation over daily spend

Revenue lives in the `subscriptions` table owned by the billing service.
"""

import os
import statistics
from datetime import datetime, timezone

import clickhouse_connect
import psycopg
from fastapi import FastAPI

app = FastAPI(title="Margin Analytics", version="0.2.0")


def _clickhouse():
    return clickhouse_connect.get_client(
        host=os.getenv("CLICKHOUSE_HOST", "localhost"),
        port=int(os.getenv("CLICKHOUSE_HTTP_PORT", "8123")),
    )


def _postgres():
    return psycopg.connect(
        os.getenv(
            "POSTGRES_DSN",
            "postgres://marginpilot:marginpilot@localhost:5432/marginpilot?sslmode=disable",
        )
    )


@app.get("/healthz")
def healthz() -> dict:
    return {"status": "ok"}


@app.get("/v1/margin/{tenant_id}")
def margin(tenant_id: str) -> dict:
    """Per-customer gross margin for the current month."""
    period = int(datetime.now(timezone.utc).strftime("%Y%m"))

    with _postgres() as conn, conn.cursor() as cur:
        cur.execute(
            "SELECT customer_id, plan, revenue_micros FROM subscriptions WHERE tenant_id = %s",
            (tenant_id,),
        )
        subs = {r[0]: {"plan": r[1], "revenue_micros": r[2]} for r in cur.fetchall()}

    rows = _clickhouse().query(
        "SELECT customer_id, sum(cost_micros) FROM usage_events "
        "WHERE tenant_id = {t:String} AND toYYYYMM(occurred_at) = {p:UInt32} "
        "GROUP BY customer_id",
        parameters={"t": tenant_id, "p": period},
    ).result_rows
    cogs = {r[0]: int(r[1]) for r in rows}

    customers = []
    for cust, sub in subs.items():
        revenue = sub["revenue_micros"]
        cost = cogs.get(cust, 0)
        margin_pct = round((revenue - cost) / revenue * 100, 2) if revenue > 0 else None
        customers.append(
            {
                "customer_id": cust,
                "plan": sub["plan"],
                "revenue_usd": round(revenue / 1e6, 2),
                "ai_cogs_usd": round(cost / 1e6, 6),
                "gross_margin_pct": margin_pct,
            }
        )
    return {"tenant_id": tenant_id, "period": period, "customers": customers}


@app.get("/v1/anomalies/{tenant_id}")
def anomalies(tenant_id: str) -> dict:
    """Flag days where a customer's spend exceeds mean + 3*std of their history."""
    rows = _clickhouse().query(
        "SELECT customer_id, toDate(occurred_at) AS d, sum(cost_micros) "
        "FROM usage_events WHERE tenant_id = {t:String} "
        "GROUP BY customer_id, d ORDER BY customer_id, d",
        parameters={"t": tenant_id},
    ).result_rows

    series: dict[str, list[tuple[str, int]]] = {}
    for cust, day, micros in rows:
        series.setdefault(cust, []).append((str(day), int(micros)))

    flagged = []
    for cust, points in series.items():
        values = [v for _, v in points]
        if len(values) < 3:
            continue
        mean = statistics.mean(values)
        std = statistics.pstdev(values)
        if std <= 0:
            continue
        for day, value in points:
            if value > mean + 3 * std:
                flagged.append(
                    {
                        "customer_id": cust,
                        "day": day,
                        "spend_usd": round(value / 1e6, 6),
                        "baseline_usd": round(mean / 1e6, 6),
                    }
                )
    return {"tenant_id": tenant_id, "anomalies": flagged}
