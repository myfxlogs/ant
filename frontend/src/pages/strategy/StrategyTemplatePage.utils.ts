
export const isTerminalRun = (run: { status?: unknown; isTerminal?: boolean; is_terminal?: boolean }) => {
  return Boolean(run?.isTerminal || run?.is_terminal);
};

export const isSucceededRun = (run: { status?: unknown; isSucceeded?: boolean; is_succeeded?: boolean }) => {
  return Boolean(run?.isSucceeded || run?.is_succeeded);
};
