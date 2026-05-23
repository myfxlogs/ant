# AntTrader MCP Server

AntTrader MCP Server exposes a narrow tool surface for agent workflows. It calls AntTrader backend through ConnectRPC only and does not read the database, credentials, or trading adapters directly.

## Configuration

- `ANTTRADER_CONNECT_URL`: backend ConnectRPC base URL, default `http://127.0.0.1:8080`
- `ANTTRADER_ACCESS_TOKEN`: user JWT or agent token used as `Authorization: Bearer ...`

## Commands

```bash
npm install
npm run build
npm start
```

## Tools

- `whoami`
- `list_accounts`
- `list_symbols`
- `get_quotes`
- `list_templates`
- `get_template`
- `submit_backtest`
- `get_backtest_job`
- `detect_market_regime`

All high-risk operations remain backend-authorized and must pass existing ConnectRPC authentication, permission, RiskEngine, and service checks.
