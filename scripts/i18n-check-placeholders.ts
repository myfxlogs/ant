#!/usr/bin/env tsx
import * as fs from 'fs';
import * as path from 'path';

const PROTO_DIR = '/opt/ant/proto/ant/v1/i18n';

function parseTextproto(content: string): Map<string, string> {
  const fields = new Map<string, string>();
  for (const line of content.split('\n')) {
    const m = line.match(/^(\w+):\s*'(.*)'\s*$/);
    if (m) fields.set(m[1], m[2].replace(/\\'/g, "'"));
  }
  return fields;
}

const enFields = parseTextproto(fs.readFileSync(path.join(PROTO_DIR, 'base_en.textproto'), 'utf-8'));

for (const locale of ['zh-cn', 'zh-tw', 'ja', 'vi']) {
  const fields = parseTextproto(fs.readFileSync(path.join(PROTO_DIR, `base_${locale}.textproto`), 'utf-8'));
  let sameAsEn = 0;
  const sameKeys: string[] = [];
  for (const [key, value] of fields) {
    const enValue = enFields.get(key);
    if (enValue && enValue === value) {
      sameAsEn++;
      sameKeys.push(key);
    }
  }
  console.log(`\n${locale}: ${sameAsEn} keys still identical to English (out of ${fields.size} total)`);
  // Show first 10
  if (sameKeys.length > 0) {
    console.log(`  Examples: ${sameKeys.slice(0, 10).join(', ')}`);
  }
}
