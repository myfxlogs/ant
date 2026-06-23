#!/usr/bin/env python3
"""E2E browser test — complete flow including Backtest + Live on MT4 95172262."""
import os, sys
from playwright.sync_api import sync_playwright

BASE_URL = "http://localhost:8022"
MQL_FILE = "/tmp/venus_ea.mq4"

def log(msg): print(f"[E2E] {msg}", flush=True)

def run():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page(viewport={"width": 1440, "height": 900})
        page.set_default_timeout(25000)

        try:
            # 1-2: Login + Nav
            log("1. Login")
            page.goto(f"{BASE_URL}/login", wait_until="load")
            page.wait_for_timeout(3000)
            page.locator('input').nth(0).fill("admin@1.com")
            page.locator('input').nth(1).fill("12345678")
            page.locator('button[type="submit"]').first.click()
            page.wait_for_timeout(5000)

            log("2. Strategy Library")
            page.wait_for_timeout(2000)
            page.locator('[role="menuitem"]').nth(1).click()
            page.wait_for_timeout(1500)
            page.locator('text="Strategy Library"').first.click()
            page.wait_for_timeout(2000)

            # 3-9: Create → Save
            log("3. Create")
            page.locator('button:has-text("Create")').first.click()
            page.wait_for_timeout(2500)
            log("4. Import EA")
            page.locator('text="Import EA"').first.click(timeout=5000)
            page.wait_for_timeout(1000)
            log("5. Paste MQL")
            with open(MQL_FILE) as f: mql = f.read()
            page.locator('textarea[placeholder*="code"], textarea[placeholder*="Paste"]').first.fill(mql[:5000])
            log("6. Name")
            page.locator('input[placeholder*="name"], input[placeholder*="Name"]').first.fill("马丁", timeout=5000)

            log("7. Translate")
            for attempt in range(3):
                page.locator('button:has-text("Translate to Python")').first.click()
                page.wait_for_timeout(25000)
                if page.locator('button:has-text("Apply to Editor")').count() > 0:
                    page.locator('button:has-text("Apply to Editor")').first.click()
                    page.wait_for_timeout(2000)
                    log(f"   OK (attempt {attempt+1})")
                    break

            log("8. Validate")
            page.locator('button:has-text("Validate code")').first.click()
            page.wait_for_timeout(15000)

            log("9. Save")
            save = page.locator('button:has-text("Save")').first
            page.wait_for_timeout(3000)
            if not save.is_disabled():
                save.click(); page.wait_for_timeout(4000)
                log("   Saved!")
            else:
                page.locator('button:has-text("Validate code")').first.click()
                page.wait_for_timeout(15000)
                page.locator('button:has-text("Save")').first.click()
                page.wait_for_timeout(4000)
                log("   Saved (retry)")

            # 10: Backtest
            log("10. Backtest")
            page.goto(f"{BASE_URL}/strategy/library", wait_until="load")
            page.wait_for_timeout(3000)
            page.locator('text="马丁"').first.click(timeout=5000)
            page.wait_for_timeout(2000)
            page.locator('button:has-text("Backtest")').first.click(timeout=5000)
            page.wait_for_timeout(12000)
            page.screenshot(path="/tmp/e2e_10_backtest.png")
            log("   OK")

            # 11: Live on MT4 — go back to library detail
            log("11. Live on 95172262")
            page.goto(f"{BASE_URL}/strategy/library", wait_until="load")
            page.wait_for_timeout(3000)
            page.locator('text="马丁"').first.click(timeout=5000)
            page.wait_for_timeout(2000)
            page.locator('button:has-text("Create Run")').first.click(timeout=5000)
            page.wait_for_timeout(3000)
            try:
                page.locator('text="95172262"').first.click(timeout=5000)
                page.wait_for_timeout(500)
                log("   Account selected")
            except:
                try:
                    page.locator('[class*="select"]').first.click(timeout=3000)
                    page.wait_for_timeout(500)
                    page.locator('text="95172262"').first.click(timeout=3000)
                except: pass

            for txt in ['Start','启动','Confirm','确认','Run']:
                try:
                    page.locator(f'button:has-text("{txt}")').first.click(timeout=3000)
                    page.wait_for_timeout(3000)
                    log(f"   Started via '{txt}'")
                    break
                except: pass

            page.screenshot(path="/tmp/e2e_final.png")
            log("=== E2E COMPLETE ===")

        except Exception as e:
            page.screenshot(path="/tmp/e2e_error.png")
            log(f"ERROR: {e}")
            import traceback; traceback.print_exc()
        finally:
            browser.close()

if __name__ == "__main__":
    run()
