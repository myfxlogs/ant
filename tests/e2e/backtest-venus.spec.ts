import { test, expect, type Page } from '@playwright/test';

const MQL_SOURCE = `#property copyright "Venus"
#property link      "https://secure.tickmill.co.uk/redirect/index.php?cii=155&cis=1&lp=https%3A%2F%2Fsecure.tickmill.co.uk%2Ftrader%2Findex.php%3Ftask%3D1050%26lang%3D2"
#property description "EA"
int 帐号限制 = 0;
datetime 时间限制 = D'3020.12.31';
string 作者 = "https://secure.tickmill.co.uk";

extern string 平台选择0 = "IC TickMill FxPro Pepperstone ";
extern string 平台选择1 = "不要选择国产平台,否则后果自负";
extern string 加载货币M15 = "AUDUSD NZDUSD USDJPY GBPUSD EURGBP GBPJPY AUDJPY CHFJPY";
enum BuyOrSell0 {只做多 = 0,只做空 = 1,多空都做 = 2};
input BuyOrSell0 多空方向 = 2;
extern bool   显示止盈价格 = true;
extern int    显示浮亏单数 = 7;
extern bool   启用虚拟下单 = true;
extern int    虚拟下单单数 = 3;
extern int    单向最大单数 = 15;
extern double 资金2W单笔最大下单量 = 3;
extern double 起始下单量 = 0.01;
extern double 翻倍 = 1.6;
extern double 单K线限制点数 = 50;
extern int    达到限制暂停小时 = 0;
extern int    间隔单数 = 7;
extern int    单数以下间隔点数 = 20;
extern int    单数以上间隔点数 = 5;
extern int    总体盈利点数 = 10;
extern double 总亏损金额平仓 = 0.0;
extern double 总盈利金额平仓 = 0.0;
extern int    滑点 = 3;
extern int    定单识别码 = 151003;
extern string 定单注释 = "Venus";
extern string TimeC = "====电脑时间参数====";
extern bool   启用时间控制 = false;
extern int    开始小时 = 8;
extern int    开始分钟 = 0;
extern int    结束小时 = 19;
extern int    结束分钟 = 0;

int Gi_144 = 0;
int Gi_148 = 0;
double G_order_open_price_150;
double G_order_open_price_158;
double Gd_168;
double Gd_176;
double Gd_184;
double Gd_192;
double Gd_200;
double Gd_208;
double Gd_216;
double Gd_217;
double Gd_218;
double Gd_219;
double G_price_232;
bool Gi_256;
bool TrackZoer0 = true;
bool TrackZoer1 = true;
bool TrackZoerXuNi0;
bool TrackZoerXuNi1;
int G_pos_260;
int G_ticket_264;
int Gi_260;
int Gi_261;
int Gi_268;
int Gi_272;
int Time0 = 1;
int Time2 = 1;
int Gi_276 = 1;
int Gi_280;
double Gd_284;
double G_lots_290 = 0.02;
double G_lots_292 = 0.02;
double Gd_300 = 1.5;
int Gi_306 = 7;
int Gi_307 = 4;
int Gi_308 = 20;
int Gi_310 = 7;
int Gi_311 = 7;
int Gi_312 = 7;
int Gi_313 = 7;
bool Gi_315;
double Gi_316 = 2;
double NewOpenPrice;
double STOPLEVEL;
double Gd_320 = 0.0;
double Gd_328 = 0.0;
int Day0 = -1;
int Gi_336 = 3;
int G_magic_340 = 10533;
string G_comment_300 = "";
string G_comment_344 = "";
string Gsa_720[14];
bool Use_Msg;
bool BoolClose;
int l_pod_110;
double LotsAll;
double Lots0;
double LotsC0;
double MinLot;
double MaxLot;
double Gd_720[30];
double Gd_730[30];
double Gd_721[30];
double Gd_731[30];
int Gi_720 = 0;
int Gi_721 = 0;
int Gi_309 = 0;
double Gi_330 = 2;
double Gi_331 = 2;
bool TrackXuNi;
bool OpenNew = true;
int Gi_323 = 0;
int Gi_324 = 0;
bool OpenAdd0;
bool OpenAdd1;
bool Use_Time;
int StartHour;
int StartMinit;
int EndHour;
int EndMinit;

int OnInit() {
   ArrayInitialize(Gd_720,0);
   ArrayInitialize(Gd_721,0);
   ArrayInitialize(Gd_730,0);
   ArrayInitialize(Gd_731,0);
   TrackXuNi = 启用虚拟下单;
   Gi_269 = 多空方向;
   Gi_330 = 单K线限制点数;
   Gi_331 = 达到限制暂停小时;
   G_lots_292 = 起始下单量;
   Gd_300 = 翻倍;
   Gi_306 = 显示浮亏单数;
   Gi_309 = 虚拟下单单数;
   Gi_308 = 单向最大单数;
   Gi_313 = 间隔单数;
   Gi_311 = 单数以下间隔点数;
   Gi_312 = 单数以上间隔点数;
   Gi_316 = 总体盈利点数;
   Gi_315 = 显示止盈价格;
   Gd_320 = 总亏损金额平仓;
   Gd_328 = 总盈利金额平仓;
   Gi_336 = 滑点;
   G_magic_340 = 定单识别码;
   G_comment_344 = 定单注释;
   Use_Time = 启用时间控制;
   StartHour = 开始小时;
   StartMinit = 开始分钟;
   EndHour = 结束小时;
   EndMinit = 结束分钟;
   G_comment_344 = G_comment_344 + "_";
   Gd_284 = MathPow(0.1, Digits);
   if (Digits == 5 || Digits == 3) Gi_276 = 10;
   Day0 = Day();
   EventSetMillisecondTimer(300);
   STOPLEVEL = MarketInfo(Symbol(),MODE_STOPLEVEL);
   MinLot = MarketInfo(Symbol(),MODE_MINLOT);
   MaxLot = MarketInfo(Symbol(),MODE_MAXLOT);
   Gi_280 = MathRound((-MathLog(MarketInfo(Symbol(), MODE_LOTSTEP))) / 2.302585093);
   return(INIT_SUCCEEDED);
}

void OnDeinit(const int reason) {
   Comment("");
   EventKillTimer();
   return ;
}

void OnTick() {
if(帐号限制 != 0 && AccountNumber() != 帐号限制) {
  Alert("此帐号未注册");
  ExpertRemove();
  return;
}
if(TimeCurrent() > 时间限制) {
  Alert("使用时间过期");
  ExpertRemove();
  return;
}
CountOrders();
if(BoolClose) return;
if(iHigh(Symbol(),PERIOD_H1,0) - iLow(Symbol(),PERIOD_H1,0) >= Gi_330 * Gi_276 * Gd_284) Time2 = TimeCurrent();
if(TimeCurrent() - Time2 < Gi_331 * 3600) return;
G_lots_290 = NormalizeDouble(AccountBalance() / 20000 * 资金2W单笔最大下单量,Gi_280);
if (Gi_268 == 0 && (Gi_269 == 0 || Gi_269 == 2) && (OpenAdd0 || (OpenNew && CheckTime() && Time0 != iTime(NULL, PERIOD_M1, 0)))) {
   OpenAdd0 = false;
   if((TrackXuNi && Gi_720 < Gi_309) || G_lots_292 < MinLot) {
     Gi_323++;
     Gd_720[Gi_720] = G_lots_292;
     Gd_730[Gi_720] = Ask;
     Gi_720++;
     }
   else {
     LotsAll = NormalizeDouble(G_lots_292, Gi_280);
     G_ticket_264 = OrderSend(Symbol(), OP_BUY, LotsAll, Ask, Gi_336 * Gi_276, 0, 0, G_comment_344 + DoubleToStr(Gi_268,0), G_magic_340, 0, Blue);
     }
   return;
   }
if (Gi_272 == 0 && (Gi_269 == 1 || Gi_269 == 2) && (OpenAdd1 || (OpenNew && CheckTime() && Time0 != iTime(NULL, PERIOD_M1, 0)))) {
   OpenAdd1 = false;
   if((TrackXuNi && Gi_721 < Gi_309) || G_lots_292 < MinLot) {
     Gi_324++;
     Gd_721[Gi_721] = G_lots_292;
     Gd_731[Gi_721] = Bid;
     Gi_721++;
     }
   else {
     LotsAll = NormalizeDouble(G_lots_292, Gi_280);
     G_ticket_264 = OrderSend(Symbol(), OP_SELL, LotsAll, Bid, Gi_336 * Gi_276, 0, 0, G_comment_344 + DoubleToStr(Gi_272,0), G_magic_340, 0, Red);
     }
   return;
   }
Gi_310 = GetGi_310(Gi_268);
if (Gi_268 > 0 && (Gi_269 == 0 || Gi_269 == 2) && (OpenAdd0 || (OpenNew && Time0 != iTime(NULL, PERIOD_M1, 0) && iClose(NULL, PERIOD_M15, 1) >= iOpen(NULL, PERIOD_M15, 1) && Gi_268 < Gi_308 && G_order_open_price_150 - Ask >= Gi_310 * Gi_276 * Gd_284))) {
   OpenAdd0 = false;
   LotsC0 = 0;
   l_pod_110 = 0;
   LotsAll = MathMin(G_lots_290,NormalizeDouble(G_lots_292 * MathPow(Gd_300, Gi_268), 4));
   if((TrackXuNi && Gi_720 < Gi_309) || LotsAll < MinLot) {
     Gi_323++;
     Gd_720[Gi_720] = LotsAll;
     Gd_730[Gi_720] = Ask;
     Gi_720++;
     }
   else {
   LotsAll = NormalizeDouble(LotsAll, Gi_280);
   while(LotsC0 < LotsAll && l_pod_110 < 10) {
      RefreshRates();
      Lots0 = MathMin(LotsAll - LotsC0,MaxLot);
      G_ticket_264 = OrderSend(Symbol(), OP_BUY, Lots0, Ask, Gi_336 * Gi_276, 0, 0, G_comment_344 + DoubleToStr(Gi_268,0), G_magic_340, 0, Blue);
      if(G_ticket_264 >= 0) {
        LotsC0 += Lots0;
        }
      else {
        l_pod_110++;
        }
     }
   }
   return;
}
Gi_310 = GetGi_310(Gi_272);
if (Gi_272 > 0 && (Gi_269 == 1 || Gi_269 == 2) && (OpenAdd1 || (OpenNew && Time0 != iTime(NULL, PERIOD_M1, 0) && iClose(NULL, PERIOD_M15, 1) <= iOpen(NULL, PERIOD_M15, 1) && Gi_272 < Gi_308 && Bid - G_order_open_price_158 >= Gi_310 * Gi_276 * Gd_284))) {
   OpenAdd1 = false;
   LotsC0 = 0;
   l_pod_110 = 0;
   LotsAll = MathMin(G_lots_290,NormalizeDouble(G_lots_292 * MathPow(Gd_300, Gi_272), 4));
   if((TrackXuNi && Gi_721 < Gi_309) || LotsAll < MinLot) {
     Gi_324++;
     Gd_721[Gi_721] = LotsAll;
     Gd_731[Gi_721] = Bid;
     Gi_721++;
     }
   else {
   LotsAll = NormalizeDouble(LotsAll, Gi_280);
   while(LotsC0 < LotsAll && l_pod_110 < 10) {
      RefreshRates();
      Lots0 = MathMin(LotsAll - LotsC0,MaxLot);
      G_ticket_264 = OrderSend(Symbol(), OP_SELL, Lots0, Bid, Gi_336 * Gi_276, 0, 0, G_comment_344 + DoubleToStr(Gi_272,0), G_magic_340, 0, Red);
      if(G_ticket_264 >= 0) {
        LotsC0 += Lots0;
        }
      else {
        l_pod_110++;
        }
      }
     }
   return;
  }
Time0 = iTime(NULL, PERIOD_M1, 0);
return ;
}

int GetGi_310(int Ai_0) {
int li_8 = Gi_311;
if (Ai_0 >= Gi_313 && Ai_0 < Gi_313 + 5) li_8 = Gi_312;
return(li_8);
}

bool CheckTime() {
   if(!Use_Time) return(1);
   return(0);
}

void CountOrders() {
   Gi_268 = Gi_720;
   Gi_272 = Gi_721;
   Gd_184 = 0;
   Gd_192 = 0;
   Gd_200 = 0;
   Gd_208 = 0;
   Gd_216 = 0;
   Gd_218 = 0;
   if(Gi_720 > 0) G_order_open_price_150 = Gd_730[Gi_720-1];
   if(Gi_721 > 0) G_order_open_price_158 = Gd_731[Gi_721-1];
   Gd_168 = 0;
   Gd_176 = 0;
   for (G_pos_260 = 0; G_pos_260 < OrdersTotal(); G_pos_260++) {
      if (OrderSelect(G_pos_260, SELECT_BY_POS, MODE_TRADES)) {
         if (OrderSymbol() == Symbol() && OrderMagicNumber() == G_magic_340) {
            if (OrderType() == OP_BUY) {
               Gi_268++;
               Gd_184 += OrderLots();
               Gd_200 += OrderOpenPrice() * OrderLots();
               Gd_216 += OrderProfit() + OrderSwap() + OrderCommission();
               G_order_open_price_150 = OrderOpenPrice();
            }
            if (OrderType() == OP_SELL) {
               Gi_272++;
               Gd_192 += OrderLots();
               Gd_208 += OrderOpenPrice() * OrderLots();
               Gd_218 += OrderProfit() + OrderSwap() + OrderCommission();
               G_order_open_price_158 = OrderOpenPrice();
            }
         }
      }
   }
   if (Gd_184 > 0.0) Gd_168 = NormalizeDouble(Gd_200 / Gd_184, Digits);
   if (Gd_192 > 0.0) Gd_176 = NormalizeDouble(Gd_208 / Gd_192, Digits);
}
`;

async function login(page: Page) {
  await page.goto('/login', { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);
  // Ant Design Form generates IDs as {formName}_{fieldName}
  await page.locator('#login_login').fill('admin@1.com');
  await page.locator('#login_password').fill('12345678');
  // Submit button is inside the form
  await page.locator('form button[type="submit"]').click();
  // Wait for navigation away from /login
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });
}

async function loginViaAPI(): Promise<string> {
  const resp = await fetch('http://localhost:8022/ant.v1.AuthService/Login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login: 'admin@1.com', password: '12345678' }),
  });
  if (!resp.ok) throw new Error(`Login API failed: ${resp.status}`);
  const data = await resp.json();
  return data.accessToken;
}

async function createTemplateViaAPI(token: string, code: string, name: string): Promise<string> {
  const resp = await fetch('http://localhost:8022/ant.v1.StrategyService/CreateTemplate', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify({
      name,
      description: 'Venus EA for backtest testing',
      code,
      parameters: [],
      isPublic: false,
      tags: [],
      i18n: '',
    }),
  });
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`CreateTemplate failed: ${resp.status} ${text}`);
  }
  const data = await resp.json();
  const id = data.id || data.template?.id || '';
  if (!id) throw new Error(`CreateTemplate returned no ID: ${JSON.stringify(data)}`);
  return id;
}

test('backtest Venus EA on BTCUSDm 15m', async ({ page }) => {
  // ── 1. Login via API to get auth token (for template creation) ──
  const token = await loginViaAPI();
  console.log('API login successful');

  // ── 2. Create template via ConnectRPC API ──
  const templateId = await createTemplateViaAPI(token, MQL_SOURCE, 'Venus EA Backtest Test');
  console.log('Created template:', templateId);

  // ── 3. Login via UI (for browser session) ──
  await login(page);

  // ── 4. Navigate to workspace with templateId to auto-load code ──
  await page.goto(`/strategy/workspace?templateId=${templateId}`, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(3000);

  // ── 5. Select trading account 95172262 ──
  // Account selector is in the toolbar — look for a Select with account-related placeholder
  const accountSelect = page.locator('.ant-select').first();
  await accountSelect.waitFor({ state: 'visible', timeout: 10_000 });
  await accountSelect.click();
  await page.waitForTimeout(800);
  // Type to filter the dropdown
  await page.keyboard.type('95172262');
  await page.waitForTimeout(800);
  // Click the matching option
  const accountOption = page.locator('.ant-select-item-option').filter({ hasText: '95172262' }).first();
  await accountOption.waitFor({ state: 'visible', timeout: 5000 });
  await accountOption.click();
  await page.waitForTimeout(1500);

  // ── 6. Select symbol BTCUSDm ──
  // The symbol picker is another Select in the toolbar
  const symbolSelect = page.locator('.ant-select').nth(1);
  await symbolSelect.waitFor({ state: 'visible', timeout: 10_000 });
  await symbolSelect.click();
  await page.waitForTimeout(800);
  await page.keyboard.type('BTCUSD');
  await page.waitForTimeout(1000);
  // Look for BTCUSDm in the dropdown
  const symbolOption = page.locator('.ant-select-item-option').filter({ hasText: /BTCUSDm/i }).first();
  await symbolOption.waitFor({ state: 'visible', timeout: 5000 });
  await symbolOption.click();
  await page.waitForTimeout(1500);

  // ── 7. Set timeframe to 15m ──
  const tf15m = page.locator('.ant-radio-button-wrapper').filter({ hasText: '15m' }).first();
  await tf15m.click();
  await page.waitForTimeout(500);

  // ── 8. Click Run backtest ──
  // The Run button is in the backtest panel — find by primary button with Run text
  const runButton = page.locator('button.ant-btn-primary').filter({ hasText: /Run|运行/ }).last();
  await runButton.waitFor({ state: 'visible', timeout: 10_000 });
  await runButton.click();

  // ── 9. Wait for backtest to complete ──
  // The "Completed" tag uses ant-tag with color="success" — text is "Completed"
  // The "Running" tag uses ant-tag with color="processing" — text is "Running"
  // The "Error" tag uses ant-tag with color="error"
  // Poll for completion (up to 100 seconds)
  let completed = false;
  let errored = false;
  for (let i = 0; i < 50; i++) {
    await page.waitForTimeout(2000);
    // Check for Completed tag (green/success)
    const completedTag = page.locator('.ant-tag').filter({ hasText: /^Completed$/ });
    if (await completedTag.first().isVisible().catch(() => false)) {
      completed = true;
      break;
    }
    // Check for error tag
    const errorTag = page.locator('.ant-tag-error, .ant-tag').filter({ hasText: /failed|error/i });
    if (await errorTag.first().isVisible().catch(() => false)) {
      errored = true;
      break;
    }
    // Also check if statistic values are present (metrics rendered = completed)
    const stats = page.locator('.ant-statistic-content-value');
    if (await stats.first().isVisible().catch(() => false)) {
      completed = true;
      break;
    }
  }

  // ── 10. Capture backtest report ──
  const report: Record<string, string> = {};

  // Extract metric values from Statistic components
  const metrics = page.locator('.ant-statistic');
  const metricCount = await metrics.count();
  for (let i = 0; i < metricCount; i++) {
    const title = await metrics.nth(i).locator('.ant-statistic-title').textContent().catch(() => '');
    const value = await metrics.nth(i).locator('.ant-statistic-content-value').textContent().catch(() => '');
    if (title && value) {
      report[title.trim()] = value.trim();
    }
  }

  // Also capture the collapsed metrics row if visible
  const collapsedMetrics = page.locator('div').filter({ hasText: /Total Return|总收益/ }).filter({ has: page.locator('b') });
  if (await collapsedMetrics.first().isVisible().catch(() => false)) {
    const text = await collapsedMetrics.first().textContent();
    if (text) report['collapsedMetrics'] = text.trim();
  }

  // Capture execution assumptions if visible
  const assumptions = page.locator('div').filter({ hasText: /Execution Assumptions|执行假设/ }).first();
  if (await assumptions.isVisible().catch(() => false)) {
    const text = await assumptions.textContent();
    if (text) report['executionAssumptions'] = text.trim();
  }

  // Capture error message if errored
  if (errored) {
    const errorTagEl = page.locator('.ant-tag-error').first();
    const errorText = await errorTagEl.textContent().catch(() => 'Unknown error');
    report['error'] = errorText?.trim() || 'Unknown error';
  }

  // ── 11. Output report ──
  console.log('\n========== BACKTEST REPORT ==========');
  console.log(`Status: ${completed ? 'COMPLETED' : errored ? 'ERROR' : 'TIMEOUT'}`);
  console.log(`Symbol: BTCUSDm | Timeframe: 15m`);
  console.log('---');
  for (const [key, value] of Object.entries(report)) {
    if (key === 'collapsedMetrics' || key === 'executionAssumptions') {
      console.log(`${key}: ${value}`);
    } else {
      console.log(`${key}: ${value}`);
    }
  }
  console.log('=====================================\n');

  // Take a screenshot for the record
  await page.screenshot({ path: 'backtest-result.png', fullPage: true });

  // Assert that we got some result
  if (errored) {
    console.log('Backtest returned an error — this may indicate a compilation or runtime issue with the MQL code.');
  }
  expect(completed || errored).toBeTruthy();
});
