// codeValidator.ts — 前端沙箱代码校验
//
// 规则镜像 backend/internal/service/debate_v2_prompts.go 里的
// `[Hard sandbox constraints]`。生成代码若不满足这些规则，
// 前端禁止"保存为模板"，并把违规项作为反馈送回 LLM 重写。

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
	// 原文（英文）消息，面向 LLM 的反馈；i18n 翻译由展示层用 t(`ai.debate.v2.validation.codes.${code}`) 负责。
	message: string;
	// 命中的字面量（可选），例如 "open" / "import pandas" / "__import__"，用于在 Alert 里高亮。
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

// 每行拆分时先去掉 # 注释和三引号字符串里的内容，避免注释里的关键字误判。
function stripPythonComments(src: string): string {
	// 删除三引号字符串（简化处理：贪婪匹配 """...""" 与 '''...'''）。
	let s = src.replace(/"""[\s\S]*?"""/g, '""')
	            .replace(/'''[\s\S]*?'''/g, "''");
	// 删除单行 # 注释。
	s = s.split('\n').map((line) => {
		// 保留引号内的 #。这里做的是快速判断，不用完整 tokenizer。
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

/** 检查 Python 代码是否满足 AntTrader 的沙箱约束。 */
export function validatePythonSandbox(raw: string): Violation[] {
	const violations: Violation[] = [];
	const code = String(raw || '');
	if (!code.trim()) {
		violations.push({ code: 'empty', message: 'The generated code is empty.' });
		return violations;
	}

	// 1. 代码块里不能再嵌套 ``` 围栏（模型若这么写说明输出被污染）。
	if (code.includes('```')) {
		violations.push({ code: 'fence_inside', message: 'The code body still contains ``` fences.' });
	}

	const stripped = stripPythonComments(code);
	const lines = stripped.split('\n');

	// 2. 禁止 import。
	const importRe = /^\s*(?:import\s+\S+|from\s+\S+\s+import\s+)/m;
	const imMatch = importRe.exec(stripped);
	if (imMatch) {
		violations.push({
			code: 'import',
			message: 'Python `import` statements are not allowed in the sandbox.',
			hit: imMatch[0].trim(),
		});
	}

	// 3. 禁止 dunder（__xxx__ 形式的访问）。
	const dunderRe = /__[A-Za-z_]+__/;
	const duMatch = dunderRe.exec(stripped);
	if (duMatch) {
		violations.push({
			code: 'dunder',
			message: `Dunder attribute access is not allowed (hit: ${duMatch[0]}).`,
			hit: duMatch[0],
		});
	}

	// 4. 禁止 banned identifier 作为"调用"形式：`name(` 或 `= name`。
	//    放宽判断：只要作为一个独立单词出现即视为命中，避免模型偷用 builtins。
	for (const id of BANNED_IDENTIFIERS) {
		const re = new RegExp(`(^|[^A-Za-z0-9_])${id}\\s*\\(`);
		if (re.test(stripped)) {
			violations.push({
				code: 'banned_identifier',
				message: `Calling the banned builtin \`${id}(...)\` is not allowed.`,
				hit: id,
			});
			break; // 报一条即可
		}
	}

	// 5. 禁止第三方包（即便漏过了 import 过滤，裸用也要拦）。
	//    注意这里匹配 ` pandas.` / `numpy(` 等调用形式，避免对注释/字符串误判。
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

	// 6. 必须存在 top-level `def run(context):`。
	const runSigRe = /^def\s+run\s*\(\s*context\s*\)\s*:/m;
	const anyRunRe = /^def\s+run\s*\(/m;
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
	// 忽略 lines 便于 TS 不告警
	void lines;

	return violations;
}

/** 把违规项序列化为中文/英文混排的反馈，送回 LLM 作为 rejectCode 的 feedback。 */
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
