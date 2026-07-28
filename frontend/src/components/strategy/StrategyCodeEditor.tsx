/**
 * StrategyCodeEditor — CodeMirror 6 wrapper for MQL strategy code.
 *
 * Replaces the plain <Input.TextArea> in WorkspaceCodePanel with syntax
 * highlighting, line numbers, bracket matching, and auto-indentation.
 * Uses C/C++ syntax highlighting since MQL is C-like.
 */
import { useEffect, useRef, useMemo } from 'react';
import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightSpecialChars, drawSelection, rectangularSelection } from '@codemirror/view';
import { EditorState, Compartment } from '@codemirror/state';
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
import { cpp } from '@codemirror/lang-cpp';
import { syntaxHighlighting, defaultHighlightStyle, bracketMatching, indentOnInput } from '@codemirror/language';
import { closeBrackets, autocompletion } from '@codemirror/autocomplete';

export interface Diagnostic {
  line: number;    // 1-based line number
  message: string;
  severity: 'error' | 'warning' | 'info';
}

interface Props {
  value: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
  diagnostics?: Diagnostic[];
  style?: React.CSSProperties;
}

const readOnlyCompartment = new Compartment();

export default function StrategyCodeEditor({ value, onChange, readOnly, _diagnostics, style }: Props) {
  const editorRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  // Build extensions once. readOnly is handled via compartment reconfigure in a separate effect.
  const extensions = useMemo(() => [
    lineNumbers(),
    highlightActiveLine(),
    highlightSpecialChars(),
    drawSelection(),
    rectangularSelection(),
    bracketMatching(),
    closeBrackets(),
    autocompletion(),
    indentOnInput(),
    cpp(),
    syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
    history(),
    keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
    readOnlyCompartment.of(EditorState.readOnly.of(!!readOnly)),
    EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        onChangeRef.current(update.state.doc.toString());
      }
    }),
    EditorView.theme({
      '&': { backgroundColor: '#1e1e2e', color: '#cdd6f4', fontSize: '13px' },
      '.cm-gutters': { backgroundColor: '#181825', color: '#6c7086', border: 'none' },
      '.cm-activeLineGutter': { backgroundColor: '#313244' },
      '.cm-activeLine': { backgroundColor: '#31324444' },
      '.cm-cursor': { borderLeftColor: '#f5e0dc' },
      '.cm-selectionBackground': { backgroundColor: '#45475a88' },
      '.cm-matchingBracket': { backgroundColor: '#45475a66', outline: '1px solid #89b4fa' },
      '&.cm-focused .cm-matchingBracket': { backgroundColor: '#45475a88' },
      '.cm-tooltip': { backgroundColor: '#313244', border: '1px solid #45475a', color: '#cdd6f4' },
    }, { dark: true }),
    EditorView.baseTheme({
      '&.cm-editor': { maxHeight: '100%' },
      '.cm-scroller': { fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace" },
    }),
  // eslint-disable-next-line react-hooks/exhaustive-deps -- readOnly handled via compartment reconfigure
  ], []);

  // Diagnostics — reserved for future @codemirror/lint integration.
  // Currently the parent WorkspaceCodePanel renders validation results
  // in an Alert below the editor, which covers the same need.

  // Create / destroy editor.
  useEffect(() => {
    if (!editorRef.current) return;
    const view = new EditorView({
      doc: value,
      extensions,
      parent: editorRef.current,
    });
    viewRef.current = view;
    return () => view.destroy();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Update readOnly compartment when prop changes.
  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    view.dispatch({
      effects: readOnlyCompartment.reconfigure(EditorState.readOnly.of(!!readOnly)),
    });
  }, [readOnly]);

  // Sync external value changes (e.g. template load, AI apply).
  // Preserve cursor/scroll position to avoid jarring jumps.
  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    const current = view.state.doc.toString();
    if (value !== current) {
      const pos = view.state.selection.main.head;
      const scroll = view.scrollDOM.scrollTop;
      view.dispatch({
        changes: { from: 0, to: current.length, insert: value },
        selection: { anchor: Math.min(pos, value.length) },
      });
      requestAnimationFrame(() => { view.scrollDOM.scrollTop = Math.min(scroll, view.scrollDOM.scrollHeight); });
    }
  }, [value]);

  return (
    <div
      ref={editorRef}
      style={{
        border: '1px solid #313244',
        borderRadius: 6,
        overflow: 'hidden',
        minHeight: 420,
        ...style,
      }}
    />
  );
}
