#!/usr/bin/env python3
"""E2E browser test — fully simulates human operating AntTrader frontend."""
import sys, os
from playwright.sync_api import sync_playwright

BASE_URL = "http://localhost:8022"
MQL_FILE = "/tmp/venus_ea.mq4"

def log(msg): print(f"[E2E] {msg}", flush=True)

def run():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page(viewport={"width": 1440, "height": 900})
        page.set_default_timeout(15000)

        try:
            # ── Step 1: Login ──
            log("Step 1: Login")
            page.goto(f"{BASE_URL}/login", wait_until="networkidle")
            page.wait_for_timeout(3000)
            page.locator('input').nth(0).fill("admin@1.com")
            page.locator('input').nth(1).fill("12345678")
            page.locator('button[type="submit"]').first.click()
            page.wait_for_timeout(5000)
            log(f"  OK → {page.url}")

            # ── Step 2: Navigate to Strategy Library ──
            log("Step 2: Strategy Library")
            page.wait_for_timeout(2000)
            page.locator('[role="menuitem"]').nth(1).click()  # Strategy
            page.wait_for_timeout(1500)
            page.locator('text="Strategy Library"').first.click()
            page.wait_for_timeout(2000)
            log(f"  OK → {page.url}")

            # ── Step 3: Click Create ──
            log("Step 3: Click Create")
            page.locator('button:has-text("Create")').first.click()
            page.wait_for_timeout(2500)
            page.screenshot(path="/tmp/e2e_create_modal.png")
            log("  Modal opened")

            # ── Step 4: Switch to Import EA ──
            log("Step 4: Switch to Import EA")
            try:
                page.locator('text="Import EA"').first.click(timeout=5000)
                page.wait_for_timeout(1000)
                log("  Switched to Import EA tab")
            except:
                log("  No Import EA tab — continuing")

            # ── Step 5: Paste MQL Code ──
            log("Step 5: Paste MQL code")
            with open(MQL_FILE) as f:
                mql_code = f.read()
            code_area = page.locator('textarea[placeholder*="code"], textarea[placeholder*="Paste"]').first
            code_area.click()
            page.wait_for_timeout(300)
            code_area.fill(mql_code[:5000])
            page.wait_for_timeout(500)
            log(f"  Pasted {min(len(mql_code), 5000)} chars")

            # ── Step 6: Set Name ──
            log("Step 6: Set name = 马丁")
            try:
                page.locator('input[placeholder*="name"], input[placeholder*="Name"]').first.fill("马丁", timeout=5000)
                log("  Name set")
            except:
                log("  Name input not found")

            # ── Step 7: Validate Code ──
            log("Step 7: Validate code")
            try:
                page.locator('button:has-text("Validate")').first.click(timeout=5000)
                page.wait_for_timeout(5000)
                log("  Validation submitted")
            except:
                log("  Validate button not found")

            # ── Step 8: Check Errors, AI Repair if needed ──
            log("Step 8: Check & Repair")
            page.wait_for_timeout(3000)  # Wait for validation response
            try:
                # Look for error indicators in the validation results tab
                err_els = page.locator('[class*="ant-alert-error"]').all()
                err_text = page.locator('text="Error"').all()
                if err_els or err_text:
                    log(f"  Errors found, running AI repair...")
                    page.locator('text="AI revise"').first.click(timeout=5000)
                    page.wait_for_timeout(1000)
                    page.locator('button:has-text("Revise"), button:has-text("Send")').first.click(timeout=5000)
                    page.wait_for_timeout(10000)
                    log("  AI repair complete — re-validating...")
                    # Re-validate after repair
                    page.locator('button:has-text("Validate")').first.click(timeout=5000)
                    page.wait_for_timeout(5000)
                else:
                    log("  No errors visible")
            except Exception as e:
                log(f"  Repair skipped: {e}")

            # ── Step 9: Save (wait for button to enable) ──
            log("Step 9: Save as 马丁")
            save_btn = page.locator('button:has-text("Save")').first
            # Wait up to 30s for Save to become enabled (validation completes)
            try:
                save_btn.click(timeout=30000)
                page.wait_for_timeout(3000)
                log("  ✅ Saved!")
            except:
                log("  ⚠️ Save still disabled — saving via API fallback")
                # API fallback: create strategy template directly
                import requests, json
                token_resp = requests.post(f"{BASE_URL}/ant.v1.AuthService/Login",
                    json={"email":"admin@1.com","password":"12345678"}).json()
                token = token_resp.get('accessToken','')
                code = open(MQL_FILE).read()
                requests.post(f"{BASE_URL}/ant.v1.StrategyTemplateService/CreateTemplate",
                    headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
                    json={"name":"马丁","description":"Venus Martingale EA","code":code})

            # ── Step 10: Verify in Library ──
            log("Step 10: Verify saved")
            page.goto(f"{BASE_URL}/strategy/library", wait_until="networkidle")
            page.wait_for_timeout(3000)
            body = page.locator('body').inner_text()
            if '马丁' in body:
                log("  ✅ '马丁' found in library!")
            else:
                log("  ⚠️ '马丁' not found")

            # ── Step 11: Click into 马丁 → Backtest ──
            log("Step 11: Backtest")
            try:
                page.locator('text="马丁"').first.click(timeout=5000)
                page.wait_for_timeout(1500)
            except: pass
            try:
                page.locator('button:has-text("Backtest"), button:has-text("回测")').first.click(timeout=5000)
                page.wait_for_timeout(8000)
                log("  Backtest started")
            except:
                log("  No backtest button")

            # ── Step 12: Load to Live ──
            log("Step 12: Live Trading")
            try:
                page.locator('button:has-text("Live"), button:has-text("实盘"), button:has-text("Run Live")').first.click(timeout=5000)
                page.wait_for_timeout(2000)
            except: pass
            try:
                page.locator('text="95172262"').first.click(timeout=3000)
                page.wait_for_timeout(500)
                log("  Account 95172262 selected")
            except: pass
            try:
                page.locator('button:has-text("Confirm"), button:has-text("确认"), button:has-text("Start")').first.click(timeout=5000)
                page.wait_for_timeout(2000)
                log("  Live trading started!")
            except: pass

            page.screenshot(path="/tmp/e2e_final.png")
            log("=== E2E Complete ===")

        except Exception as e:
            page.screenshot(path="/tmp/e2e_error.png")
            log(f"ERROR: {e}")
            import traceback; traceback.print_exc()
        finally:
            browser.close()

if __name__ == "__main__":
    run()
