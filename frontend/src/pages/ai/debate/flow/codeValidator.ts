// codeValidator.ts — Frontend sandbox code validation
//
// Rules mirror backend/internal/service/debate_v2_prompts.go
// [Hard sandbox constraints]. If generated code violates these rules,
// the frontend blocks "save as template" and sends violations back to LLM for rewrite.

export type ViolationCode =
	| 'import'
	| 'dunder'
	| 'banned_identifier'
	| 'forbidden_package'
	| 'missing_run'
	| 'wrong_signature'
	| 'fence_inside'
	| 'empty';

export interface Violation {
	code: ViolationCode;
	// Original (English) messages for LLM feedback; i18n translation handled by display layer via t(`ai.debate.v2.validation.codes.${code}`).
	message: string;
	// Matched literal (optional), e.g. "open" / "import pandas" / "__import__", used for highlighting in Alert.
	hit?: string;
}

const BANNED_IDENTIFIERS = [
	'open', 'eval', 'exec', 'compile', 'globals', 'locals',
	'vars', 'dir', 'input', 'breakpoint', 'help', 'exit', 'quit',
	'getattr', 'setattr', 'delattr', 'hasattr',
];

const FORBIDDEN_PACKAGES = [
	'pandas', 'numpy', 'backtrader', 'ccxt', 'requests',
	'os', 'sys', 'subprocess', 'pathlib', 'pickle', 'socket',
	'shutil', 'multiprocessing', 'threading', 'asyncio',
];

// Strip # comments and triple-quoted strings before line analysis to avoid false positives.
function stripPythonComments(src: string): string {
	// Remove triple-quoted strings (simplified: greedy match for """...""" and '''...''').
	let s = src.replace(/"""[\s\S]*?"""/g, '""')
	            .replace(/'''[\s\S]*?'''/g, "''");
	// Remove single-line # comments.
	s = s.split('\n').map((line) => {
		// Preserve # inside quotes. This is a fast heuristic, not a full tokenizer.
		let inS = false;
		let inD = false;
		for (let i = 0; i < line.length; i++) {
			const c = line[i];
			if (c === '\\') { i++; continue; }
			if (!inD && c === '\'') inS = !inS;
			else if (!inS && c === '"') inD = !inD;
			else if (!inS && !inD && c === '#') return line.slice(0, i);
		}
		return line;
	}).join('\n');
	return s;
}

/** Check if Python code satisfies AntTrader sandbox constraints. */
export function validatePythonSandbox(raw: string): Violation[] {
	const violations: Violation[] = [];
	const code = String(raw || '');
	if (!code.trim()) {
		violations.push({ code: 'empty', message: 'The generated code is empty.' });
		return violations;
	}

	// 1. No nested ``` fences inside code block (indicates corrupted model output).
	if (code.includes('```')) {
		violations.push({ code: 'fence_inside', message: 'The code body still contains ``` fences.' });
	}

	const stripped = stripPythonComments(code);
	const lines = stripped.split('\n');

	// 2. No import statements.
	const importRe = /^\s*(?:import\s+\S+|from\s+\S+\s+import\s+)/m;
	const imMatch = importRe.exec(stripped);
	if (imMatch) {
		violations.push({
			code: 'import',
			message: 'Python `import` statements are not allowed in the sandbox.',
			hit: imMatch[0].trim(),
		});
	}

	// 3. No dunder access (__xxx__ patterns).
	const dunderRe = /__[A-Za-z_]+__/;
	const duMatch = dunderRe.exec(stripped);
	if (duMatch) {
		violations.push({
			code: 'dunder',
			message: `Dunder attribute access is not allowed (hit: ${duMatch[0]}).`,
			hit: duMatch[0],
		});
	}

	// 4. No banned identifiers used as call patterns: name( or = name.
	//    Relaxed: any standalone occurrence is flagged to prevent builtin abuse.
	for (const id of BANNED_IDENTIFIERS) {
		const re = new RegExp(`(^|[^A-Za-z0-9_])${id}\\s*\\(`);
		if (re.test(stripped)) {
			violations.push({
				code: 'banned_identifier',
				message: `Calling the banned builtin \`${id}(...)\` is not allowed.`,
				hit: id,
			});
			break; // Report one violation per category
		}
	}

	// 5. No third-party packages (even if import filter missed, bare usage is blocked).
	//    Matches call patterns like pandas. / numpy(, avoiding comment/string false positives.
	for (const pkg of FORBIDDEN_PACKAGES) {
		const re = new RegExp(`(^|[^A-Za-z0-9_])${pkg}\\s*[\\.\\(]`);
		if (re.test(stripped)) {
			violations.push({
				code: 'forbidden_package',
				message: `Third-party / system package \`${pkg}\` is not available in the sandbox.`,
				hit: pkg,
			});
			break;
		}
	}

	// 6. Must have top-level def run(context):.
	const runSigRe = /^\s*def\s+run\s*\(\s*context\s*\)\s*:/m;
	const anyRunRe = /^\s*def\s+run\s*\(/m;
	if (!runSigRe.test(stripped)) {
		if (anyRunRe.test(stripped)) {
			const m = anyRunRe.exec(stripped);
			violations.push({
				code: 'wrong_signature',
				message: `The entry point must be \`def run(context):\` (found: ${m ? m[0] : '?'}).`,
				hit: m ? m[0] : undefined,
			});
		} else {
			violations.push({
				code: 'missing_run',
				message: 'Missing the `def run(context):` entry point at top level.',
			});
		}
	}
	// Ignore lines to satisfy TS unused-variable lint
	void lines as unknown; // suppress TS noUnusedLocals

	return violations;
}

/** Serialize violations as feedback for LLM, used as rejectCode feedback. */
export function violationsToFeedback(violations: Violation[]): string {
	if (violations.length === 0) return '';
	const head = 'The previous code failed the sandbox validator. Please rewrite so that ALL of the following are fixed:';
	const items = violations.map((v, i) => `${i + 1}. [${v.code}] ${v.message}`);
	const tail = [
		'Reminders:',
		'- The entry must be exactly `def run(context):` at top level.',
		'- Do NOT use any `import` statement, any dunder (e.g. __import__), or banned builtins (open/eval/exec/compile/globals/locals/getattr/...).',
		'- Do NOT use pandas / numpy / backtrader / os / sys / requests etc.; read everything from `context`.',
		'- Output only ONE ```python ...``` fenced block.',
	];
	return [head, ...items, '', ...tail].join('\n');
}
