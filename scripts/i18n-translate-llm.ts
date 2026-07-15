#!/usr/bin/env tsx
/**
 * Uses Anthropic Claude API to translate English placeholder entries
 * in zh-tw, ja, vi textproto files.
 *
 * Usage: npx tsx scripts/i18n-translate-llm.ts
 *
 * Environment:
 *   ANTHROPIC_AUTH_TOKEN - API key
 *   ANTHROPIC_BASE_URL   - API base URL
 *   ANTHROPIC_MODEL      - model name (default: claude-sonnet-4-20250514)
 */
import * as fs from 'fs';
import * as path from 'path';
import * as https from 'https';
import * as http from 'https';

const PROTO_DIR = '/opt/ant/proto/ant/v1/i18n';

interface TextprotoEntry {
  key: string;
  value: string;
  line: string;
  lineIndex: number;
}

function parseTextproto(content: string): TextprotoEntry[] {
  const entries: TextprotoEntry[] = [];
  const lines = content.split('\n');
  for (let i = 0; i < lines.length; i++) {
    const m = lines[i].match(/^(\w+):\s*'(.*)'\s*$/);
    if (m) {
      entries.push({
        key: m[1],
        value: m[2].replace(/\\'/g, "'"),
        line: lines[i],
        lineIndex: i,
      });
    }
  }
  return entries;
}

function callClaudeAPI(
  apiKey: string,
  baseUrl: string,
  model: string,
  systemPrompt: string,
  userMessage: string,
): Promise<string> {
  return new Promise((resolve, reject) => {
    const url = new URL(baseUrl.replace(/\/$/, '') + '/v1/messages');
    const body = JSON.stringify({
      model,
      max_tokens: 8000,
      system: systemPrompt,
      messages: [{ role: 'user', content: userMessage }],
    });

    const options = {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'x-api-key': apiKey,
        'anthropic-version': '2023-06-01',
      },
    };

    const req = (url.protocol === 'https:' ? https : http).request(url, options, (res) => {
      let data = '';
      res.on('data', (chunk) => (data += chunk));
      res.on('end', () => {
        if (res.statusCode !== 200) {
          reject(new Error(`API returned ${res.statusCode}: ${data.substring(0, 500)}`));
          return;
        }
        try {
          const json = JSON.parse(data);
          const text = json.content?.map((c: any) => c.text).join('') || '';
          resolve(text);
        } catch (e) {
          reject(new Error(`Failed to parse API response: ${e}`));
        }
      });
    });
    req.on('error', reject);
    req.write(body);
    req.end();
  });
}

const LOCALE_CONFIG: Record<string, { name: string; instructions: string }> = {
  'zh-cn': {
    name: 'Simplified Chinese (简体中文, Mainland China)',
    instructions: `Translate to Simplified Chinese (Mainland China). Rules:
- Keep technical terms like "API", "SSE", "Token", "MT4", "MT5", "gRPC", "URL", "ID" as-is
- Keep brand names like "AlphaForge", "OpenAI", "Claude", "DeepSeek" as-is
- Use Mainland-style terminology (e.g. 设置, 订阅, 策略, 账户)
- Keep placeholders like {{count}}, {{amount}}, {{name}} unchanged
- Keep interpolation markers like {defaultValue} unchanged
- Translate the meaning, not word-by-word
- For UI labels, keep it concise (usually 2-6 characters)`,
  },
  'zh-tw': {
    name: 'Traditional Chinese (繁體中文, Taiwan)',
    instructions: `Translate to Traditional Chinese (Taiwan). Rules:
- Keep technical terms like "API", "SSE", "Token", "MT4", "MT5", "gRPC", "URL", "ID" as-is
- Keep brand names like "AlphaForge", "OpenAI", "Claude", "DeepSeek" as-is
- Use Taiwan-style terminology (e.g. 策略 not 策略, 設定 not 設置, 訂閱 not 订阅)
- Keep placeholders like {{count}}, {{amount}}, {{name}} unchanged
- Keep interpolation markers like {defaultValue} unchanged
- Translate the meaning, not word-by-word
- For UI labels, keep it concise (usually 2-6 characters)`,
  },
  'ja': {
    name: 'Japanese (日本語)',
    instructions: `Translate to Japanese. Rules:
- Keep technical terms like "API", "SSE", "Token", "MT4", "MT5", "gRPC", "URL", "ID" as-is
- Keep brand names like "AlphaForge", "OpenAI", "Claude", "DeepSeek" as-is
- Use polite/formal style (です/ます) for descriptions, plain form for labels
- Keep placeholders like {{count}}, {{amount}}, {{name}} unchanged
- For UI labels, keep it concise
- Use katakana for common loan words (e.g. トークン, ウォレット, サブスクリプション)`,
  },
  'vi': {
    name: 'Vietnamese (Tiếng Việt)',
    instructions: `Translate to Vietnamese. Rules:
- Keep technical terms like "API", "SSE", "Token", "MT4", "MT5", "gRPC", "URL", "ID" as-is
- Keep brand names like "AlphaForge", "OpenAI", "Claude", "DeepSeek" as-is
- Keep placeholders like {{count}}, {{amount}}, {{name}} unchanged
- For UI labels, keep it concise
- Use natural Vietnamese phrasing, not word-by-word translation`,
  },
};

async function translateBatch(
  apiKey: string,
  baseUrl: string,
  model: string,
  locale: string,
  entries: { key: string; value: string }[],
): Promise<Map<string, string>> {
  const config = LOCALE_CONFIG[locale];
  const systemPrompt = `You are a professional UI translator. ${config.instructions}

You will receive a list of English UI strings in format "key: value". 
Return ONLY the translated strings in the same format "key: translated_value".
Do not add any explanation, markdown, or code blocks. Just the lines.`;

  // Build the user message
  const lines = entries.map(e => `${e.key}: ${e.value}`);
  const userMessage = `Translate these ${entries.length} UI strings to ${config.name}:\n\n${lines.join('\n')}`;

  const response = await callClaudeAPI(apiKey, baseUrl, model, systemPrompt, userMessage);

  // Parse response lines
  const result = new Map<string, string>();
  for (const line of response.split('\n')) {
    const m = line.match(/^(\w+):\s*(.+)$/);
    if (m) {
      result.set(m[1], m[2].trim());
    }
  }
  return result;
}

async function main() {
  const apiKey = process.env.ANTHROPIC_AUTH_TOKEN || process.env.ANTHROPIC_API_KEY;
  if (!apiKey) {
    console.error('No Anthropic API key found in environment');
    process.exit(1);
  }
  const baseUrl = process.env.ANTHROPIC_BASE_URL || 'https://api.anthropic.com';
  const model = process.env.ANTHROPIC_MODEL || 'claude-sonnet-4-20250514';

  console.log(`Using model: ${model}`);
  console.log(`Base URL: ${baseUrl}`);

  // Load English textproto as reference
  const enEntries = parseTextproto(fs.readFileSync(path.join(PROTO_DIR, 'base_en.textproto'), 'utf-8'));
  const enMap = new Map(enEntries.map(e => [e.key, e.value]));

  const BATCH_SIZE = 50; // entries per API call

  for (const locale of ['zh-cn', 'zh-tw', 'ja', 'vi']) {
    console.log(`\n=== Translating ${locale} ===`);

    const filePath = path.join(PROTO_DIR, `base_${locale}.textproto`);
    const content = fs.readFileSync(filePath, 'utf-8');
    const entries = parseTextproto(content);

    // Find entries that are identical to English (placeholders)
    const placeholders = entries.filter(e => {
      const enValue = enMap.get(e.key);
      return enValue && enValue === e.value;
    });

    // Skip keys that should remain English (brand names, language names, etc.)
    const skipKeys = new Set([
      'app_name', 'language_english', 'language_japanese', 'language_vietnamese',
      'language_chinese_simplified', 'language_chinese_traditional',
    ]);
    const toTranslate = placeholders.filter(e => !skipKeys.has(e.key));

    console.log(`  Total entries: ${entries.length}`);
    console.log(`  Identical to English: ${placeholders.length}`);
    console.log(`  Skipped (brand/lang names): ${placeholders.length - toTranslate.length}`);
    console.log(`  To translate: ${toTranslate.length}`);

    if (toTranslate.length === 0) {
      console.log('  Nothing to translate, skipping.');
      continue;
    }

    // Split into batches
    const batches: typeof toTranslate[] = [];
    for (let i = 0; i < toTranslate.length; i += BATCH_SIZE) {
      batches.push(toTranslate.slice(i, i + BATCH_SIZE));
    }

    console.log(`  Batches: ${batches.length} (size: ${BATCH_SIZE})`);

    const translations = new Map<string, string>();
    let batchNum = 0;

    for (const batch of batches) {
      batchNum++;
      process.stdout.write(`  Batch ${batchNum}/${batches.length}...`);

      try {
        const result = await translateBatch(
          apiKey,
          baseUrl, 
          model,
          locale,
          batch.map(e => ({ key: e.key, value: e.value })),
        );

        let translated = 0;
        for (const entry of batch) {
          const translatedValue = result.get(entry.key);
          if (translatedValue) {
            translations.set(entry.key, translatedValue);
            translated++;
          }
        }
        console.log(` ${translated}/${batch.length} translated`);

        // Small delay to avoid rate limiting
        await new Promise(r => setTimeout(r, 500));
      } catch (e) {
        console.error(` ERROR: ${(e as Error).message}`);
        // Continue with next batch
      }
    }

    // Apply translations to the file
    const fileLines = content.split('\n');
    let applied = 0;
    for (const entry of entries) {
      const translatedValue = translations.get(entry.key);
      if (translatedValue) {
        const escapedValue = translatedValue.replace(/'/g, "\\'");
        fileLines[entry.lineIndex] = `${entry.key}: '${escapedValue}'`;
        applied++;
      }
    }

    fs.writeFileSync(filePath, fileLines.join('\n'));
    console.log(`  Applied ${applied} translations to base_${locale}.textproto`);
  }

  console.log('\n=== Done ===');
}

main().catch(e => {
  console.error('Fatal error:', e);
  process.exit(1);
});
