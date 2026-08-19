declare module 'react-syntax-highlighter/dist/esm/index' {
  import type { ComponentType } from 'react';
  interface ReactSyntaxHighlighterProps {
    language?: string;
    style?: Record<string, unknown>;
    showLineNumbers?: boolean;
    wrapLines?: boolean;
    customStyle?: React.CSSProperties;
    lineNumberStyle?: React.CSSProperties;
    children?: string;
  }
  export const Light: ComponentType<ReactSyntaxHighlighterProps> & {
    registerLanguage: (name: string, fn: (hljs: unknown) => unknown) => void;
  };
  export const Prism: ComponentType<ReactSyntaxHighlighterProps> & {
    registerLanguage: (name: string, fn: (hljs: unknown) => unknown) => void;
  };
}

declare module 'react-syntax-highlighter/dist/esm/prism' {
  import type { ComponentType } from 'react';
  export const Prism: ComponentType<ReactSyntaxHighlighterProps>;
  interface ReactSyntaxHighlighterProps {
    language?: string;
    style?: Record<string, unknown>;
    showLineNumbers?: boolean;
    wrapLines?: boolean;
    customStyle?: React.CSSProperties;
    lineNumberStyle?: React.CSSProperties;
    children?: string;
  }
}

declare module 'react-syntax-highlighter/dist/esm/languages/hljs/python' {
  const python: (hljs: unknown) => unknown;
  export default python;
}

declare module 'react-syntax-highlighter/dist/esm/styles/hljs' {
  export const atomOneDark: Record<string, unknown>;
}

declare module 'react-syntax-highlighter/dist/esm/styles/prism' {
  export const vscDarkPlus: Record<string, unknown>;
}
