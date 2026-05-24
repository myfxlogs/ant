// Wraps a user strategy with a small Python prelude that injects the
// user-supplied parameter values into ``context['params']`` at runtime.
//
// We use this because the gRPC ``StartBacktestRunRequest`` does not (yet)
// carry a ``params`` field — rather than touch protobuf + Go + Python in
// one shot, we patch the strategy code locally on submit. The wrapper:
//
//   * Defines a private dict ``__ANTTRADER_BACKTEST_PARAMS__``.
//   * Re-binds ``run`` to a wrapper that merges this dict into
//     ``context['params']`` before delegating to the user's original
//     ``run``. Existing keys in ``context['params']`` win, so the engine
//     can still override at runtime if it ever starts populating params.
//
// The wrapper is appended (not prepended) so the user's original code
// keeps line-numbers stable — handy when sandbox errors reference
// ``line N`` from the saved template.

/**
 * Encode a JS value into a Python literal we trust. We only allow scalars
 * (string / number / boolean / null) here — required-params UI doesn't
 * collect anything richer than that today.
 */
function toPythonLiteral(v: unknown): string {
	if (v === null || v === undefined) return 'None';
	if (typeof v === 'boolean') return v ? 'True' : 'False';
	if (typeof v === 'number') {
		if (!Number.isFinite(v)) return 'None';
		return String(v);
	}
	// Strings: use JSON.stringify and rely on the fact that JSON's escape
	// rules are a strict subset of valid Python string literals.
	return JSON.stringify(String(v));
}

export function wrapStrategyCodeWithParams(code: string, params: Record<string, unknown>): string {
	const entries = Object.entries(params || {}).filter(([, v]) => v !== undefined && v !== '' && v !== null);
	if (entries.length === 0) return code;

	const dictBody = entries
		.map(([k, v]) => `    ${JSON.stringify(k)}: ${toPythonLiteral(v)},`)
		.join('\n');

	const wrapper = [
		'',
		'# anttrader: backtest params injected at submit time (do NOT edit)',
		'__ANTTRADER_BACKTEST_PARAMS__ = {',
		dictBody,
		'}',
		'try:',
		'    __anttrader_orig_run = run  # noqa: F821',
		'    def run(context):  # type: ignore[misc]',
		'        ctx = dict(context or {})',
		'        ctx["params"] = {**__ANTTRADER_BACKTEST_PARAMS__, **(ctx.get("params") or {})}',
		'        return __anttrader_orig_run(ctx)',
		'except NameError:',
		'    pass',
		'',
	].join('\n');

	return code.replace(/\s*$/, '\n') + wrapper;
}
