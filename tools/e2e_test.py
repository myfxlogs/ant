from playwright.sync_api import sync_playwright

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    page = browser.new_page(viewport={"width": 1440, "height": 900})
    page.set_default_timeout(15000)

    # Login
    page.goto("http://localhost:8022/login", wait_until="networkidle")
    page.wait_for_timeout(3000)
    page.locator('input').nth(0).fill("admin@1.com")
    page.locator('input').nth(1).fill("12345678")
    page.locator('button[type="submit"]').first.click()
    page.wait_for_timeout(5000)
    print(f"1. Login OK: {page.url}")

    # Navigate
    page.wait_for_timeout(2000)
    page.locator('[role="menuitem"]').nth(1).click()
    page.wait_for_timeout(1500)
    page.screenshot(path="/tmp/nav_after_click.png")
    print("2. Clicked Strategy")
    
    # Check what's visible now
    lib = page.locator('text="Strategy Library"')
    print(f"3. Library count: {lib.count()}")
    if lib.count() > 0:
        lib.first.click()
        page.wait_for_timeout(2000)
        print(f"4. Library OK: {page.url}")

    # Step 3: Click Create
    print(f"\n5. Looking for Create button...")
    btns = page.locator('button:visible').all()
    for b in btns:
        t = b.inner_text().strip()
        if t and len(t) < 30:
            print(f"   Button: '{t}'")
    
    create = page.locator('button:has-text("Create")')
    print(f"6. Create buttons: {create.count()}")
    if create.count() > 0:
        create.first.click()
        page.wait_for_timeout(2000)
        print(f"7. Create clicked: {page.url}")
    
    page.screenshot(path="/tmp/e2e_simple_final.png")
    browser.close()
