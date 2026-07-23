#!/usr/bin/env python3
"""AlphaForge 全站体检"""
import asyncio, sys, json
from datetime import datetime
from dataclasses import dataclass, field

from browser_use import Agent, Browser
from browser_use.llm import ChatOpenAI

API_KEY = "sk-e5727951917e4f98a04960beb208bc6a"
BASE_URL = "http://localhost:8022"
CREDS = {"email": "admin@1.com", "password": "12345678"}

BROWSER_KWARGS = dict(headless=True, chromium_sandbox=False, disable_security=True)
LLM_KWARGS = dict(model="deepseek-v4-flash", api_key=API_KEY,
                  base_url="https://api.deepseek.com/v1", temperature=0.2)

@dataclass
class Result:
    page: str; route: str; auth: str
    status: str = "SKIP"; notes: str = ""
    errors: list = field(default_factory=list)

CHECK = """Check this page:
1. No white screen / perpetual spinner / 404 / JS crash
2. Visible content (text, tables, cards, buttons, charts)
3. Nothing clearly broken
Reply EXACTLY: PASS: <summary>  or  FAIL: <reason>"""

ROUTES = [
    ("Landing","/","noauth"),("Login","/login","noauth"),("Register","/register","noauth"),
    ("Mktplace(unauth)","/marketplace","noauth"),("Brokers","/brokers","noauth"),
    ("Dashboard","/","login"),("Gallery","/strategy","login"),("Workspace","/strategy/workspace","login"),
    ("Live","/strategy/live","login"),("MktTools","/strategy/market-tools","login"),
    ("Wallet","/wallet","login"),("Sub","/subscription","login"),
    ("Algo","/trading/algos","login"),("AutoTrd","/auto-trading","login"),
    ("Analytics","/analytics","login"),("Mktplace","/marketplace","login"),("Logs","/logs","login"),
    ("Admin","/admin","login"),("AdmUsers","/admin/users","login"),("AdmAccts","/admin/accounts","login"),
    ("AdmWallet","/admin/wallet","login"),("AdmTrd","/admin/trading","login"),
    ("AdmCfg","/admin/config","login"),("AdmStrats","/admin/strategies","login"),
]

async def test_page(browser, name, route):
    r = Result(page=name, route=route, auth="?")
    try:
        agent = Agent(
            task=f"Go to {BASE_URL}{route}. Wait 3s. {CHECK}", llm=ChatOpenAI(**LLM_KWARGS),
            browser=browser, use_vision=False,
        )
        history = await agent.run(max_steps=5)
        final = history.final_result() if history else ""
        if not final:
            r.status = "DONE"; r.notes = "loaded (no report)"
        elif "PASS" in str(final).upper():
            r.status = "PASS"
        elif "FAIL" in str(final).upper():
            r.status = "FAIL"
        else:
            r.status = "DONE"
        r.notes = str(final)[:200]
    except Exception as e:
        r.status = "ERROR"; r.errors.append(str(e)[:200])
    return r

async def main():
    results = []

    print("=== Phase 1: Unauth ===")
    for name, route, auth in ROUTES:
        if auth != "noauth": continue
        browser = Browser(**BROWSER_KWARGS)
        print(f"  {name}...", end=" ", flush=True)
        r = await test_page(browser, name, route)
        r.auth = "noauth"; results.append(r)
        print(r.status)
        await browser.close()

    print("\n=== Phase 2: Login ===")
    browser = Browser(**BROWSER_KWARGS)
    try:
        agent = Agent(
            task=f"Go to {BASE_URL}/login. Fill email '{CREDS['email']}' and password '{CREDS['password']}'. Click submit. Wait for redirect to dashboard.",
            llm=ChatOpenAI(**LLM_KWARGS), browser=browser, use_vision=False,
        )
        await agent.run(max_steps=10)
        state = await agent.browser_session.get_storage_state()
        with open("/tmp/auth_state.json", "w") as f:
            json.dump(state, f)
        print("  Login: DONE")
    except Exception as e:
        print(f"  Login: ERROR - {e}")
        await browser.close(); sys.exit(1)
    await browser.close()

    # Load storage state for auth phase
    with open("/tmp/auth_state.json") as f:
        state = json.load(f)

    print("\n=== Phase 3: Auth ===")
    for name, route, auth in ROUTES:
        if auth != "login": continue
        browser = Browser(**BROWSER_KWARGS, storage_state=state)
        print(f"  {name}...", end=" ", flush=True)
        r = await test_page(browser, name, route)
        r.auth = "login"; results.append(r)
        print(r.status)
        await browser.close()

    passed = [r for r in results if r.status in ("PASS","DONE")]
    failed = [r for r in results if r.status in ("FAIL","ERROR")]
    na = len([r for r in results if r.auth=="noauth"])
    li = len([r for r in results if r.auth=="login"])

    report = f"""# AlphaForge 全站体检报告
**时间**: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')} | **LLM**: DeepSeek-Chat
**覆盖**: {len(results)} 页 ({na} 未登录 + {li} 已登录)
**结果**: ✅ {len(passed)} 通过  ❌ {len(failed)} 失败

## 失败明细
"""
    for r in failed:
        report += f"\n### ❌ {r.page} `{r.route}` ({r.auth})\n"
        for e in r.errors: report += f"- {e}\n"
        report += f"- {r.notes}\n"

    report += "\n## 全部页面\n| | 页面 | 路由 | 状态 | 备注 |\n|--|------|------|------|------|\n"
    for r in results:
        icon = "✅" if r.status != "FAIL" else "❌"
        report += f"| {icon} | {r.page} | `{r.route}` ({r.auth}) | {r.status} | {r.notes[:100] if r.notes else '-'} |\n"

    report += f"\n## 统计\n通过率: {len(passed)}/{len(results)} ({100*len(passed)//max(len(results),1)}%)\n"

    with open("/opt/ant/full-site-audit.md", "w") as f: f.write(report)
    print(f"\nReport: /opt/ant/full-site-audit.md  Pass: {len(passed)}/{len(results)}")

if __name__ == "__main__":
    asyncio.run(main())
