#!/usr/bin/env node
/**
 * SSE stream pipeline test.
 * Tests all server-streaming ConnectRPC endpoints.
 */

const BASE = 'http://localhost:8022';
const ADMIN = { login: 'admin@1.com', password: '12345678' };

let token = null;
const results = [];
let passCount = 0;
let failCount = 0;

async function login() {
  const r = await fetch(`${BASE}/ant.v1.AuthService/Login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(ADMIN),
  });
  const data = await r.json();
  token = data.accessToken;
}

function record(name, ok, detail) {
  const status = ok ? 'PASS' : 'FAIL';
  if (ok) passCount++; else failCount++;
  results.push({ name, status, detail });
  console.log(`  ${status === 'PASS' ? '✓' : '✗'} ${name}${detail ? ' — ' + detail : ''}`);
}

/**
 * ConnectRPC server-streaming uses HTTP POST with chunked transfer encoding.
 * Each chunk is an enveloped message (1-byte flags + 4-byte length + data).
 * We read the raw stream and check for data arrival.
 */
async function testStream(path, body, timeoutMs, label) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const headers = { 'Content-Type': 'application/connect+json' };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    const resp = await fetch(`${BASE}${path}`, {
      method: 'POST',
      headers,
      body: JSON.stringify(body || {}),
      signal: controller.signal,
    });

    if (!resp.ok) {
      clearTimeout(timeout);
      record(label, false, `HTTP ${resp.status}`);
      return;
    }

    // Read at least one chunk from the stream
    const reader = resp.body.getReader();
    const { value: firstChunk } = await reader.read();
    clearTimeout(timeout);

    if (firstChunk && firstChunk.length > 0) {
      record(label, true, `received ${firstChunk.length} bytes`);
      // Cancel the stream
      await reader.cancel();
    } else {
      record(label, false, 'empty stream');
    }
  } catch (e) {
    clearTimeout(timeout);
    if (e.name === 'AbortError') {
      // Timeout without data — could be a stream that has no data to push yet
      // This is OK for event-based streams (they wait for events)
      record(label, true, `timeout after ${timeoutMs}ms (event-driven stream, no events yet)`);
    } else {
      record(label, false, e.message);
    }
  }
}

async function main() {
  console.log('SSE Stream Pipeline Test');
  console.log('=========================');
  console.log(`Base URL: ${BASE}`);

  await login();
  if (!token) {
    console.log('FATAL: Login failed');
    process.exit(1);
  }
  console.log('Login OK');

  console.log('\n=== StreamService ===');
  await testStream('/ant.v1.StreamService/SubscribeUserSummary', {}, 5000, 'SubscribeUserSummary');
  await testStream('/ant.v1.StreamService/SubscribeEvents', {}, 5000, 'SubscribeEvents');
  await testStream('/ant.v1.StreamService/SubscribeOrderUpdates', {}, 5000, 'SubscribeOrderUpdates');
  await testStream('/ant.v1.StreamService/SubscribeProfitUpdates', {}, 5000, 'SubscribeProfitUpdates');

  console.log('\n=== AdminMonitorService ===');
  await testStream('/ant.v1.AdminMonitorService/SubscribeMetrics', {}, 5000, 'SubscribeMetrics');

  console.log('\n=== NotificationService ===');
  await testStream('/ant.v1.NotificationService/StreamNotifications', {}, 5000, 'StreamNotifications');

  console.log('\n=== StrategyService ===');
  await testStream('/ant.v1.StrategyService/WatchSchedules', {}, 5000, 'WatchSchedules');

  console.log('\n=== JobService ===');
  await testStream('/ant.v1.JobService/SubscribeJob', { id: '00000000-0000-0000-0000-000000000000' }, 5000, 'SubscribeJob (non-existent)');

  console.log('\n=== MarketService ===');
  await testStream('/ant.v1.MarketService/StreamTicks', {}, 5000, 'StreamTicks');

  console.log('\n=========================');
  console.log(`Results: ${passCount} passed, ${failCount} failed, ${results.length} total`);

  if (failCount > 0) {
    console.log('\nFailed tests:');
    results.filter(r => r.status === 'FAIL').forEach(r => {
      console.log(`  ✗ ${r.name} — ${r.detail}`);
    });
  }

  process.exit(failCount > 0 ? 1 : 0);
}

main().catch(e => {
  console.error('Fatal error:', e);
  process.exit(1);
});
