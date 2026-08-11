
export const isTerminalRun = (run: { status?: unknown }) => {
  return Boolean(run?.isTerminal || run?.is_terminal);
};

export const isSucceededRun = (run: { status?: unknown }) => {
  return Boolean(run?.isSucceeded || run?.is_succeeded);
};
