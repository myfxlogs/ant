/**
 * i18n-source-editor - Precise TypeScript object literal editing.
 *
 * Reads TS source text (the "const x = { ... } as const" pattern), locates
 * specific keys by their dot-separated path, and can:
 *  - replace a string value in-place (preserving all surrounding formatting)
 *  - insert a new key-value pair into a parent object (preserving indentation)
 *
 * Zero eval(), zero format loss - operates on raw text with a state-machine parser.
 *
 * IMPORTANT: No backtick characters in comments (esbuild compatibility).
 */

export interface KeyLocation {
  /** Byte offset in source where the key name starts */
  keyStart: number;
  /** Byte offset where the value starts (opening quote, brace, bracket, or digit) */
  valueStart: number;
  /** Byte offset where the value ends (after closing quote, brace, bracket, or last digit) */
  valueEnd: number;
  /** The full dot-separated path to this key */
  path: string;
  /** The string value (only valid for string-typed values; null for objects/arrays) */
  stringValue: string | null;
  /** The quote character used: single, double, backtick, or empty for non-strings */
  quote: string;
  /** List of child keys (only for object-typed values) */
  children: KeyLocation[];
  /** Parent location (only for nested keys) */
  parent: KeyLocation | null;
}

/**
 * Parse a TS "as const" object literal source and return a map from key path
 * to KeyLocation. Handles nested objects, arrays, strings (all quote types),
 * numbers, booleans, null, and single-line // comments.
 */
export function parseSource(source: string): Map<string, KeyLocation> {
  const map = new Map<string, KeyLocation>();
  const root: KeyLocation = {
    keyStart: 0, valueStart: 0, valueEnd: source.length,
    path: '', stringValue: null, quote: '', children: [], parent: null,
  };

  // Locate the object literal: first { after = to matching } before "as const"
  const eqIdx = source.indexOf('=');
  if (eqIdx === -1) return map;
  const openBrace = source.indexOf('{', eqIdx);
  if (openBrace === -1) return map;

  // State: current position in source
  let pos = 0;

  // Skip whitespace and single-line comments
  const skipGap = () => {
    while (pos < source.length) {
      const c = source[pos];
      if (c === ' ' || c === '\t' || c === '\n' || c === '\r') { pos++; continue; }
      if (c === '/' && source[pos + 1] === '/') {
        pos += 2;
        while (pos < source.length && source[pos] !== '\n') pos++;
        continue;
      }
      break;
    }
  };

  // Parse an object body starting just after the opening {
  const parseObjectBody = (pathPrefix: string, parent: KeyLocation): number => {
    while (pos < source.length) {
      skipGap();
      if (source[pos] === '}') { pos++; break; }
      if (source[pos] === ',') { pos++; skipGap(); continue; }

      // Read key name
      const keyStart = pos;
      let keyName: string;
      if (source[pos] === "'" || source[pos] === '"') {
        const q = source[pos];
        pos++;
        const nameStart = pos;
        while (pos < source.length && source[pos] !== q) {
          if (source[pos] === '\\') pos++;
          pos++;
        }
        keyName = source.slice(nameStart, pos);
        pos++; // closing quote
      } else {
        const nameStart = pos;
        while (pos < source.length && /[a-zA-Z0-9_$]/.test(source[pos])) pos++;
        keyName = source.slice(nameStart, pos);
      }

      skipGap();
      if (source[pos] === ':') pos++;
      skipGap();

      const childPath = pathPrefix ? pathPrefix + '.' + keyName : keyName;
      pos = parseValue(childPath, parent);
    }
    return pos;
  };

  // Parse a value at current position (pos)
  const parseValue = (pathPrefix: string, parent: KeyLocation): number => {
    skipGap();
    const start = pos;
    const ch = source[pos];

    // String values
    if (ch === "'" || ch === '"' || ch === '`') {
      const quote = ch;
      pos++;
      const valStart = pos;
      while (pos < source.length) {
        if (source[pos] === '\\') { pos += 2; continue; }
        if (source[pos] === quote) { break; }
        pos++;
      }
      const valEnd = pos;
      pos++; // skip closing quote
      const loc: KeyLocation = {
        keyStart: start, valueStart: valStart, valueEnd: valEnd,
        path: pathPrefix, stringValue: source.slice(valStart, valEnd),
        quote, children: [], parent,
      };
      parent.children.push(loc);
      map.set(pathPrefix, loc);
      return pos;
    }

    // Object literal
    if (ch === '{') {
      const objStart = pos;
      pos++; // skip {
      const objLoc: KeyLocation = {
        keyStart: start, valueStart: objStart, valueEnd: 0,
        path: pathPrefix, stringValue: null, quote: '', children: [], parent,
      };
      parent.children.push(objLoc);
      map.set(pathPrefix, objLoc);
      pos = parseObjectBody(pathPrefix, objLoc);
      objLoc.valueEnd = pos;
      return pos;
    }

    // Array literal - skip to matching ]
    if (ch === '[') {
      const arrStart = pos;
      pos++;
      let depth = 1;
      let inStr = false;
      let strCh = '';
      while (pos < source.length && depth > 0) {
        const c = source[pos];
        if (inStr) {
          if (c === '\\') { pos += 2; continue; }
          if (c === strCh) { inStr = false; }
        } else {
          if (c === "'" || c === '"' || c === '`') { inStr = true; strCh = c; }
          else if (c === '[') depth++;
          else if (c === ']') depth--;
        }
        pos++;
      }
      const loc: KeyLocation = {
        keyStart: start, valueStart: arrStart, valueEnd: pos,
        path: pathPrefix, stringValue: null, quote: '', children: [], parent,
      };
      parent.children.push(loc);
      map.set(pathPrefix, loc);
      return pos;
    }

    // Number, boolean, null, undefined
    {
      const valStart = pos;
      while (pos < source.length && !/[,}\s]/.test(source[pos]) && source[pos] !== '\n') pos++;
      const loc: KeyLocation = {
        keyStart: start, valueStart: valStart, valueEnd: pos,
        path: pathPrefix, stringValue: source.slice(valStart, pos),
        quote: '', children: [], parent,
      };
      parent.children.push(loc);
      map.set(pathPrefix, loc);
      return pos;
    }
  };

  pos = openBrace;
  parseValue('', root);
  return map;
}

/**
 * Get the original source text for the value at a key location.
 */
export function getValueSource(source: string, loc: KeyLocation): string {
  if (loc.stringValue !== null) {
    return loc.quote + loc.stringValue + loc.quote;
  }
  return source.slice(loc.valueStart, loc.valueEnd);
}

/**
 * Replace the VALUE of a key in source, returning the modified source.
 * Only works for string-typed values. Preserves quote style.
 */
export function replaceValue(
  source: string,
  loc: KeyLocation,
  newStringValue: string,
): string {
  if (loc.stringValue === null) {
    throw new Error('Cannot replace non-string value at ' + loc.path);
  }

  // Choose appropriate quote
  let quote = loc.quote;
  let escaped = newStringValue;

  // Escape for the chosen quote type
  const hasNewline = newStringValue.includes('\n');
  const hasSingle = newStringValue.includes("'");
  const hasBacktick = newStringValue.includes('`');
  const hasDollar = newStringValue.includes('${');

  if (hasNewline || (hasSingle && hasBacktick) || hasDollar) {
    // Must use backtick template literal
    quote = '`';
    escaped = newStringValue
      .replace(/\\/g, '\\\\')
      .replace(/`/g, '\\`')
      .replace(/\${/g, '\\${');
  } else if (hasSingle) {
    // Use double quotes
    quote = '"';
    escaped = newStringValue
      .replace(/\\/g, '\\\\')
      .replace(/"/g, '\\"');
  } else {
    // Use single quotes (safest, most common in the codebase)
    quote = "'";
    escaped = newStringValue
      .replace(/\\/g, '\\\\')
      .replace(/'/g, "\\'");
  }

  // Build: ...quote + oldValue + quote... -> ...quote + newValue + quote...
  // For strings, valueStart points to first char INSIDE quotes
  // So the opening quote is at valueStart - 1, closing at valueEnd + 1
  const before = source.slice(0, loc.valueStart - 1);
  const after = source.slice(loc.valueEnd + 1);
  return before + quote + escaped + quote + after;
}

/**
 * Insert a new key-value pair into an object. parentLoc is the KeyLocation
 * of the parent OBJECT. The new key is inserted before the parent's closing }.
 */
export function insertKey(
  source: string,
  parentLoc: KeyLocation,
  keyName: string,
  stringValue: string,
): string {
  // Find the closing } of the parent object
  const closeBrace = parentLoc.valueEnd - 1;

  // Determine indentation from last child or by computing from parent
  let indent = '  ';
  if (parentLoc.children.length > 0) {
    const lastChild = parentLoc.children[parentLoc.children.length - 1];
    const lineStart = source.lastIndexOf('\n', lastChild.keyStart) + 1;
    indent = source.slice(lineStart, lastChild.keyStart);
  } else {
    const parentLineStart = source.lastIndexOf('\n', parentLoc.valueStart) + 1;
    const parentIndent = source.slice(parentLineStart, parentLoc.valueStart);
    indent = parentIndent + '  ';
  }

  // Choose quote for the new value
  const hasNewline = stringValue.includes('\n');
  const hasSingle = stringValue.includes("'");
  let quote = "'";
  let escaped = stringValue;
  if (hasNewline || (hasSingle && stringValue.includes('`'))) {
    quote = '`';
    escaped = stringValue
      .replace(/\\/g, '\\\\')
      .replace(/`/g, '\\`')
      .replace(/\${/g, '\\${');
  } else if (hasSingle) {
    quote = '"';
    escaped = stringValue.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
  } else {
    escaped = stringValue.replace(/\\/g, '\\\\').replace(/'/g, "\\'");
  }

  const hasExistingChildren = parentLoc.children.length > 0;

  if (hasExistingChildren) {
    // Insert before the closing brace, with a comma after the previous item
    // Find position just before the closing brace (skip trailing whitespace/newlines)
    let insertPos = closeBrace;
    while (insertPos > 0 && /[\s\n]/.test(source[insertPos - 1])) insertPos--;

    const before = source.slice(0, insertPos);
    const after = source.slice(insertPos);
    const newEntry = ',\n' + indent + keyName + ': ' + quote + escaped + quote;
    return before + newEntry + '\n' + after.trimStart();
  } else {
    // First child - insert after the opening brace
    const openBrace = parentLoc.valueStart;
    const beforeBrace = source.slice(0, openBrace + 1);
    const afterBrace = source.slice(openBrace + 1);
    // afterBrace starts with whitespace, then }
    const wsMatch = afterBrace.match(/^(\s*)/);
    const ws = wsMatch ? wsMatch[1] : '';
    const rest = afterBrace.slice(ws.length);
    return beforeBrace + ws + '\n' + indent + keyName + ': ' + quote + escaped + quote + '\n' + rest;
  }
}
