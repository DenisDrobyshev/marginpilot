# analytics (Python)

Margin analytics and ML service. Owns the "why" behind the numbers:

- per-customer margin: plan revenue (PostgreSQL) vs AI COGS (ClickHouse)
- spend forecasting and cost-anomaly detection (the ML surface of the platform)
- feeds the Notifier when a customer's spend trend threatens margin

## Run locally

```bash
cd services/analytics
python -m venv .venv && . .venv/bin/activate   # Windows: .venv\Scripts\activate
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8000
# GET http://localhost:8000/v1/margin/demo-tenant
```
