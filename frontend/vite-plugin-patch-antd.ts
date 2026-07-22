import type { Plugin } from 'vite';

/**
 * Patches antd's Dropdown to handle array children in React 19.
 *
 * React 19's React.Children.only() requires a single React element.
 * antd v6.5.1's Dropdown isPrimitive check handles primitives but not arrays.
 * This adds Array.isArray() to wrap array children in <span>.
 */
export function patchAntdDropdown(): Plugin {
  return {
    name: 'patch-antd-dropdown',
    renderChunk(code, chunk) {
      if (!chunk.fileName.includes('vendor-antd')) return null;

      const target = '.Children.only(isPrimitive(children)';
      const replacement = '.Children.only((isPrimitive(children)||Array.isArray(children))';

      if (code.includes(target)) {
        const patched = code.replace(target, replacement);
        console.log(`[patch-antd] Patched (${code.length} -> ${patched.length} bytes)`);
        return { code: patched, map: null };
      }
      return null;
    },
  };
}
