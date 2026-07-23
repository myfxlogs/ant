// check-wallet-i18n.ts
import * as fs from 'fs';
import * as path from 'path';

const ROOT = path.resolve('.');
const files = [
  'frontend/src/pages/wallet/WalletPage.tsx',
  'frontend/src/pages/wallet/PasskeyManagement.tsx',
  'frontend/src/pages/wallet/WithdrawalPanel.tsx',
  'frontend/src/pages/wallet/WhitelistManagement.tsx',
  'frontend/src/components/wallet/WalletDropdown.tsx',
];

async function main() {
  const enMod = await import(path.join(ROOT, 'frontend/src/i18n/resources/en/index.ts'));
  const enObj = enMod.default || enMod;

  function keyExists(obj: any, key: string): boolean {
    const parts = key.split('.');
    let cur = obj;
    for (const p of parts) {
      if (cur === null || cur === undefined || typeof cur !== 'object') return false;
      if (!(p in cur)) return false;
      cur = cur[p];
    }
    return typeof cur === 'string';
  }

  const keys = new Set<string>();
  for (const f of files) {
    const content = fs.readFileSync(path.join(ROOT, f), 'utf-8');
    const re = /\bt\(\s*['"]([a-zA-Z0-9_.]+)['"]\s*(?:,|\))/g;
    let m;
    while ((m = re.exec(content)) !== null) keys.add(m[1]);
  }

  const missing = [...keys].filter(k => k.startsWith('wallet.')).sort().filter(k => !keyExists(enObj, k));
  console.log(`Missing wallet keys (${missing.length}):`);
  for (const k of missing) console.log(`  ${k}`);
}

main().catch(console.error);
