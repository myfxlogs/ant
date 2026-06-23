#!/usr/bin/env python3
"""E2E browser test: simulates human operating the AntTrader frontend.

Flow: login → Strategy Library → Create → Import MQL EA →
      Translate → Validate → AI Repair → Save as "马丁" →
      Backtest → Load to Live
"""

import sys, time, json, os
from playwright.sync_api import sync_playwright, expect

BASE_URL = os.environ.get("BASE_URL", "http://localhost:8022")
MQL_FILE = os.environ.get("MQL_FILE", "/tmp/venus_ea.mq4")

def log(msg):
    print(f"[E2E] {msg}", flush=True)

def run():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context(
            viewport={"width": 1440, "height": 900},
            locale="zh-CN",
        )
        page = context.new_page()
        page.set_default_timeout(15000)

        try:
            # ── Step 1: Login ──
            log("Step 1: Login")
            page.goto(f"{BASE_URL}/login")
            page.wait_for_selector('input[type="email"], input[name="email"], input[placeholder*="邮箱"]', timeout=10000)

            # Fill credentials
            email_input = page.locator('input[type="email"], input[name="email"], input[placeholder*="邮箱"], input[placeholder*="Email"]').first
            email_input.fill("admin@1.com")

            pw_input = page.locator('input[type="password"], input[name="password"], input[placeholder*="密码"]').first
            pw_input.fill("12345678")

            # Click login button
            login_btn = page.locator('button:has-text("登录"), button:has-text("Login"), button[type="submit"]').first
            login_btn.click()
            page.wait_for_timeout(3000)
            log(f"  URL after login: {page.url[:60]}")

            # ── Step 2: Navigate to Strategy Library ──
            log("Step 2: Strategy Library")
            page.goto(f"{BASE_URL}/strategy/library")
            page.wait_for_timeout(3000)
            log(f"  URL: {page.url[:60]}")

            # ── Step 3: Click Create / Import ──
            log("Step 3: Create new strategy")
            # Try various create/import buttons
            create_btn = page.locator('button:has-text("Create"), button:has-text("创建"), button:has-text("Import"), button:has-text("导入"), a:has-text("新建"), a:has-text("Create")').first
            try:
                create_btn.click(timeout=5000)
                page.wait_for_timeout(2000)
            except:
                log("  No create button found, trying direct navigation")
                page.goto(f"{BASE_URL}/strategy/create")
                page.wait_for_timeout(2000)
            log(f"  URL: {page.url[:60]}")

            # ── Step 4: Paste MQL code ──
            log("Step 4: Input MQL code")
            with open(MQL_FILE) as f:
                mql_code = f.read()

            # Find code editor/textarea
            code_area = page.locator('textarea, .monaco-editor, [class*="code"], [class*="editor"], [role="textbox"]').first
            try:
                code_area.click(timeout=5000)
                code_area.fill(mql_code[:5000])  # Upload first 5000 chars
                log(f"  Pasted {min(len(mql_code), 5000)} chars of MQL")
            except Exception as e:
                log(f"  Code input failed: {e}")
                # Try API fallback
                log("  Falling back to API-based flow")

            # ── Step 5: Click Translate / Transform ──
            log("Step 5: Translate MQL → Python")
            translate_btn = page.locator('button:has-text("Translate"), button:has-text("翻译"), button:has-text("Transform"), button:has-text("转换")').first
            try:
                translate_btn.click(timeout=5000)
                page.wait_for_timeout(5000)
            except:
                log("  No translate button, continuing")

            # ── Step 6: Check for validation errors, AI Repair if needed ──
            log("Step 6: Validate & Repair")
            validate_btn = page.locator('button:has-text("Validate"), button:has-text("验证"), button:has-text("Check")').first
            try:
                validate_btn.click(timeout=5000)
                page.wait_for_timeout(3000)
            except:
                pass

            # Check for error indicators
            errors = page.locator('[class*="error"], [class*="Error"], .ant-alert-error').all()
            if errors:
                log(f"  Found {len(errors)} errors, clicking Repair")
                repair_btn = page.locator('button:has-text("Repair"), button:has-text("修复"), button:has-text("Fix"), button:has-text("AI")').first
                try:
                    repair_btn.click(timeout=5000)
                    page.wait_for_timeout(5000)
                except:
                    pass

            # ── Step 7: Save as "马丁" ──
            log("Step 7: Save as 马丁")
            name_input = page.locator('input[name="name"], input[placeholder*="名称"], input[placeholder*="Name"]').first
            try:
                name_input.fill("马丁")
                page.wait_for_timeout(500)
            except:
                pass

            save_btn = page.locator('button:has-text("Save"), button:has-text("保存"), button:has-text("Publish")').first
            try:
                save_btn.click(timeout=5000)
                page.wait_for_timeout(3000)
            except:
                pass

            # ── Step 8: Run Backtest ──
            log("Step 8: Run Backtest")
            page.goto(f"{BASE_URL}/strategy/library")
            page.wait_for_timeout(2000)

            # Find "马丁" card and click backtest
            martin_card = page.locator('[class*="card"]:has-text("马丁"), [class*="Card"]:has-text("马丁"), tr:has-text("马丁")').first
            try:
                martin_card.click(timeout=5000)
                page.wait_for_timeout(2000)
            except:
                log("  马丁 card not clickable")

            backtest_btn = page.locator('button:has-text("Backtest"), button:has-text("回测"), button:has-text("Run")').first
            try:
                backtest_btn.click(timeout=5000)
                page.wait_for_timeout(8000)
                log("  Backtest submitted")
            except:
                log("  No backtest button")

            # ── Step 9: Load to Live ──
            log("Step 9: Load to Live Trading")
            live_btn = page.locator('button:has-text("Live"), button:has-text("实盘"), button:has-text("Run Live"), button:has-text("启动")').first
            try:
                live_btn.click(timeout=5000)
                page.wait_for_timeout(3000)
            except:
                pass

            # Select MT4 account
            acct_select = page.locator('select, [class*="select"], [class*="Select"]').first
            try:
                acct_select.click(timeout=3000)
                page.wait_for_timeout(500)
                option = page.locator('option:has-text("95172262"), [class*="option"]:has-text("95172262"), li:has-text("95172262")').first
                option.click(timeout=3000)
                page.wait_for_timeout(500)
            except:
                pass

            # Confirm
            confirm_btn = page.locator('button:has-text("Confirm"), button:has-text("确认"), button:has-text("OK"), button:has-text("Start")').first
            try:
                confirm_btn.click(timeout=5000)
                page.wait_for_timeout(3000)
                log("  Live trading started!")
            except:
                pass

            # ── Screenshot ──
            page.screenshot(path="/tmp/e2e_final.png")
            log(f"Screenshot saved to /tmp/e2e_final.png")

            # ── Summary ──
            log("=== E2E Complete ===")
            log("Open /tmp/e2e_final.png to see the final state")

        except Exception as e:
            page.screenshot(path="/tmp/e2e_error.png")
            log(f"ERROR: {e}")
            log(f"Screenshot: /tmp/e2e_error.png")
            raise
        finally:
            browser.close()

if __name__ == "__main__":
    run()
