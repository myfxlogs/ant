// Overrides MQL extern/input parameter default values in strategy source code.
//
// MQL parameters are declared as:
//   extern int Lots = 1;
//   input double StopLoss = 50.0;
//   input bool EnableVirtual = true;
//
// This function replaces the default value in each declaration with the
// user-supplied value from the backtest params panel.

function formatMQLValue(v: unknown): string {
	if (v === null || v === undefined) return '';
	if (typeof v === 'boolean') return v ? 'true' : 'false';
	if (typeof v === 'number') return String(v);
	return String(v);
}

export function wrapStrategyCodeWithParams(code: string, params: Record<string, unknown>): string {
	const entries = Object.entries(params || {}).filter(([, v]) => v !== undefined && v !== '' && v !== null);
	if (entries.length === 0) return code;

	let result = code;
	for (const [name, value] of entries) {
		const formatted = formatMQLValue(value);
		if (!formatted) continue;
		// Match: extern <type> <name> = <default>;
		// Match: input <type> <name> = <default>;
		// Replace the default value while preserving the declaration.
		const escapedName = name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
		const re = new RegExp(
			`(\\b(?:extern|input)\\s+\\w+\\s+${escapedName}\\s*=\\s*)[^;]+`,
			'g'
		);
		result = result.replace(re, `$1${formatted}`);
	}
	return result;
}
