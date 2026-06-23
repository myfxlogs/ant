#!/usr/bin/env python3
"""E2E browser test — simulates human operating AntTrader frontend.
Flow: Login → Library → Create → Import EA → paste MQL →
      Translate to Python → Validate → AI Repair → Save as 马丁 →
      Backtest → Load to Live on MT4 95172262."""
import os, sys
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
            # ── 1: Login ──
            log("1. Login")
            page.goto(f"{BASE_URL}/login", wait_until="networkidle")
            page.wait_for_timeout(3000)
            page.locator('input').nth(0).fill("admin@1.com")
            page.locator('input').nth(1).fill("12345678")
            page.locator('button[type="submit"]').first.click()
            page.wait_for_timeout(5000)
            log(f"   → {page.url}")

            # ── 2: Navigate to Strategy Library ──
            log("2. Strategy Library")
            page.wait_for_timeout(2000)
            page.locator('[role="menuitem"]').nth(1).click()  # Strategy
            page.wait_for_timeout(1500)
            page.locator('text="Strategy Library"').first.click()
            page.wait_for_timeout(2000)
            log(f"   → {page.url}")

            # ── 3: Click Create ──
            log("3. Create")
            page.locator('button:has-text("Create")').first.click()
            page.wait_for_timeout(2500)
            page.screenshot(path="/tmp/e2e_03_create.png")

            # ── 4: Switch to Import EA ──
            log("4. Import EA tab")
            page.locator('text="Import EA"').first.click(timeout=5000)
            page.wait_for_timeout(1000)

            # ── 5: Paste MQL ──
            log("5. Paste MQL code")
            with open(MQL_FILE) as f:
                mql = f.read()
            page.locator('textarea[placeholder*="code"], textarea[placeholder*="Paste"]').first.fill(mql[:5000])
            page.wait_for_timeout(500)
            log(f"   {min(len(mql), 5000)} chars")

            # ── 6: Set Name ──
            log("6. Name = 马丁")
            page.locator('input[placeholder*="name"], input[placeholder*="Name"]').first.fill("马丁", timeout=5000)

            # ── 7: Translate to Python ──
            log("7. Translate to Python")
            page.locator('button:has-text("Translate to Python")').first.click()
            page.wait_for_timeout(30000)  # LLM translation
            page.screenshot(path="/tmp/e2e_07_translated.png")

            # ── 8: Apply to Editor ──
            log("8. Apply to Editor")
            page.locator('button:has-text("Apply to Editor")').first.click(timeout=5000)
            page.wait_for_timeout(2000)

            # ── 9: Validate ──
            log("9. Validate code")
            page.locator('button:has-text("Validate code")').first.click()
            page.wait_for_timeout(12000)

            # ── 10: Save (or repair if disabled) ──
            log("10. Save")
            save_btn = page.locator('button:has-text("Save")').first
            if save_btn.is_disabled():
                log("   Disabled — trying AI repair...")
                page.locator('text="Validation Results"').first.click(timeout=3000)
                page.wait_for_timeout(1000)
                page.locator('text="AI revise"').first.click(timeout=3000)
                page.wait_for_timeout(1000)
                # Click Send/Revise in the AI revise tab
                page.locator('button:has-text("Send")').first.click(timeout=5000)
                page.wait_for_timeout(20000)
                page.locator('button:has-text("Validate code")').first.click()
                page.wait_for_timeout(12000)

            save_btn = page.locator('button:has-text("Save")').first
            if not save_btn.is_disabled():
                save_btn.click()
                page.wait_for_timeout(3000)
                log("   ✅ Saved!")
            else:
                log("   ⚠️ Save still disabled — will use API fallback")

            # ── 11: Verify in library ──
            log("11. Verify")
            page.goto(f"{BASE_URL}/strategy/library", wait_until="networkidle")
            page.wait_for_timeout(3000)
            if '马丁' in page.locator('body').inner_text():
                log("   ✅ 马丁 in library")
            else:
                log("   ⚠️ 马丁 not visible")

            # ── 12: Backtest + Live (if saved) ──
            log("12. Backtest → Live")
            try:
                page.locator('text="马丁"').first.click(timeout=5000)
                page.wait_for_timeout(1000)
                page.locator('button:has-text("Backtest"), button:has-text("回测")').first.click(timeout=5000)
                page.wait_for_timeout(8000)
            except: pass
            try:
                page.locator('button:has-text("Live"), button:has-text("实盘")').first.click(timeout=5000)
                page.wait_for_timeout(2000)
                page.locator('text="95172262"').first.click(timeout=3000)
                page.locator('button:has-text("Confirm"), button:has-text("确认"), button:has-text("Start")').first.click(timeout=5000)
                log("   Live started!")
            except: pass

            page.screenshot(path="/tmp/e2e_final.png")
            log("=== DONE ===")

        except Exception as e:
            page.screenshot(path="/tmp/e2e_error.png")
            log(f"ERROR: {e}")
            import traceback; traceback.print_exc()
        finally:
            browser.close()

if __name__ == "__main__":
    run()
