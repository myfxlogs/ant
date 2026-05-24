import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Alert, Button, Divider, Empty, Form, Input, Modal, Space, Spin, Tag, Typography, message as antMessage } from 'antd';
import { useTranslation } from 'react-i18next';
import type { AIAgentDefinitionView } from '@/client/ai';
import { strategyApi } from '@/client/strategy';
import { CodeExplainPanel } from '@/components/strategy/CodeAssist';
import { validatePythonSandbox, violationsToFeedback, type Violation } from './flow/codeValidator';
import { useDebateFlow, type ChatMessage, type StepKey } from './flow/useDebateFlow';

/**
 * 8 个系统内置 Agent 类型：它们的展示名与描述会跟随 i18n 语系切换，
 * 读取 `ai.settings.agent.types.<type>` 与
 * `ai.settings.agent.defaults.<type>.inputHint`，而不是用户在 AI 设置里
 * 保存的固定字符串（存库值会在创建时冻结为某一语言）。
 */
const BUILTIN_AGENT_TYPES = new Set([
	'style', 'signals', 'risk', 'macro', 'sentiment', 'portfolio', 'execution', 'code',
]);

/** 返回一对 (name, hint)：内置类型优先取 i18n，自定义类型用用户存的。 */
function useAgentLabel() {
	const { t } = useTranslation();
	return (a: AIAgentDefinitionView) => {
		if (BUILTIN_AGENT_TYPES.has(a.type)) {
			const name = t(`ai.settings.agent.types.${a.type}`, { defaultValue: a.type });
			const hint = t(`ai.settings.agent.defaults.${a.type}.inputHint`, { defaultValue: '' });
			return { name, hint };
		}
		return { name: a.name || a.type, hint: a.inputHint || a.identity || '' };
	};
}

const { Text, Paragraph } = Typography;

function formatElapsed(seconds: number): string {
	const m = Math.floor(seconds / 60);
	const s = seconds % 60;
	return `${m}:${String(s).padStart(2, '0')}`;
}

/**
 * 识别"要求进入下一步"类短语——用户若发这种消息，直接触发 onNext 而不发给模型。
 * 过滤条件：长度 ≤ 16 字符，去空白和常见标点后，命中关键词集合之一。
 */
function looksLikeNextIntent(raw: string): boolean {
	const t = (raw || '').trim();
	if (!t || t.length > 16) return false;
	const stripped = t.toLowerCase().replace(/[\s。.,，！!?？~～、:：;；"'""''`]+/g, '');
	const keywords = new Set([
		'下一步', '下一个', '下一环节', '下一阶段', '下一位', '下一轮',
		'下一个agent', '下一个智能体', '下一位agent', '下一位智能体',
		'下一位专家', '下一个专家', '下一位能手', '换下一位',
		'继续', '进入下一步', '进入下一个', '进入下一环节', '进入下一阶段',
		'可以了', '可以了下一步', '好的下一步', '好了下一步', '没问题下一步',
		'ok', 'ok下一步', '好下一步', 'next', 'nextstep', 'nextagent',
		'continue', 'proceed', 'goon', 'goahead', '确定下一步', '行下一步',
	]);
	return keywords.has(stripped);
}

// -- Step: Agent Selection ----------------------------------------------------

export function AgentSelectionStep(props: {
	agentDefs: AIAgentDefinitionView[];
	agentsLoading: boolean;
	selectedAgents: AIAgentDefinitionView[];
	onChange: (agents: AIAgentDefinitionView[]) => void;
	onNext: () => void;
}) {
	const { t } = useTranslation();
	const labelOf = useAgentLabel();
	const { agentDefs, agentsLoading, selectedAgents, onChange, onNext } = props;

	// 隐藏 code 类型：代码生成器固定参与最终步骤，不需要用户勾选。
	const selectable = useMemo(() => agentDefs.filter((a) => a.type !== 'code'), [agentDefs]);

	const selectedKeys = useMemo(() => new Set(selectedAgents.map((a) => a.agentKey || a.type)), [selectedAgents]);

	function toggle(a: AIAgentDefinitionView) {
		if (!a.enabled) {
			antMessage.warning(t('ai.debate.messages.enableAgentFirst', { defaultValue: 'This expert is disabled. Please enable it in AI Settings first.' }));
			return;
		}
		const key = a.agentKey || a.type;
		if (selectedKeys.has(key)) {
			onChange(selectedAgents.filter((x) => (x.agentKey || x.type) !== key));
		} else {
			onChange([...selectedAgents, a]);
		}
	}

	return (
		<div>
			<Alert
				className="ai-gold-alert"
				type="info"
				showIcon
				style={{ marginBottom: 12 }}
				message={t('ai.debate.v2.selectTitle', { defaultValue: 'Choose the experts for this session' })}
				description={t('ai.debate.v2.selectDesc', {
					defaultValue: 'Optional: if no expert is chosen, the system will skip directly to code generation after intent clarification. The selection order is also the speaking order.',
				})}
			/>

			{agentsLoading && agentDefs.length === 0 ? (
				<div style={{ textAlign: 'center', padding: '32px 0' }}>
					<Spin />
					<div style={{ marginTop: 8 }}>
						<Text type="secondary">{t('ai.debate.messages.loadingAgents')}</Text>
					</div>
				</div>
			) : selectable.length === 0 ? (
				<Empty description={t('ai.debate.messages.noAgentsHint')} />
			) : (
				<div
					style={{
						display: 'grid',
						gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))',
						gap: 12,
					}}
				>
					{selectable.map((a) => {
						const key = a.agentKey || a.type;
						const selected = selectedKeys.has(key);
						const order = selected
							? selectedAgents.findIndex((x) => (x.agentKey || x.type) === key) + 1
							: 0;
						return (
							<div
								key={key}
								onClick={() => toggle(a)}
								style={{
									cursor: 'pointer',
									borderRadius: 8,
									border: selected
										? '2px solid #d4af37'
										: a.enabled
										? '1px solid #e5e7eb'
										: '1px solid #f0f0f0',
									background: selected
										? 'rgba(212, 175, 55, 0.08)'
										: a.enabled
										? '#ffffff'
										: '#fafafa',
									padding: 12,
									minHeight: 84,
									display: 'flex',
									flexDirection: 'column',
									gap: 6,
								}}
							>
								{(() => {
									const { name: label, hint } = labelOf(a);
									return (
										<>
											<div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
												<Text strong>{label}</Text>
												{selected ? <Tag color="gold">#{order}</Tag> : null}
											</div>
											<Text type="secondary" style={{ fontSize: 12 }}>
												{hint || label}
											</Text>
										</>
									);
								})()}
							</div>
						);
					})}
				</div>
			)}

			<div style={{ marginTop: 16, textAlign: 'right' }}>
				<Space>
					<Text type="secondary">
						{t('ai.debate.v2.selectedCount', {
							defaultValue: '{{count}} expert(s) selected',
							count: selectedAgents.length,
						})}
					</Text>
					<Button type="primary" onClick={onNext}>
						{selectedAgents.length === 0
							? t('ai.debate.v2.nextNoAgents', { defaultValue: 'No expert needed, next' })
							: t('ai.debate.v2.next', { defaultValue: 'Next' })}
					</Button>
				</Space>
			</div>
		</div>
	);
}

// -- Step: Chat (intent or per-agent) ----------------------------------------

export function ChatStep(props: {
	stepKey: StepKey;
	stepLabel: string;
	state: ReturnType<ReturnType<typeof useDebateFlow>['stepState']>;
	sending: boolean;
	modelWaitActive: boolean;
	modelWaitElapsedSeconds: number;
	/** LLM deltas while async advance (kickoff) is streaming via SSE. */
	streamingPreview?: string;
	onSend: (text: string) => Promise<void>;
	onBack: () => void;
	onNext: () => void;
	isFirstChat: boolean;
	isLastAgent: boolean;
	canBack?: boolean;
}) {
	const { t } = useTranslation();
	const {
		stepLabel,
		state,
		sending,
		modelWaitActive,
		modelWaitElapsedSeconds,
		streamingPreview,
		onSend,
		onBack,
		onNext,
		isFirstChat,
		isLastAgent,
		canBack = true,
	} = props;
	const [input, setInput] = useState('');
	const listRef = useRef<HTMLDivElement | null>(null);

	useEffect(() => {
		const el = listRef.current;
		if (el) el.scrollTop = el.scrollHeight;
	}, [state.messages.length, streamingPreview]);

	async function handleSend() {
		const text = input.trim();
		if (!text) {
			antMessage.warning(t('ai.debate.messages.inputFirst'));
			return;
		}
		// 若用户输入"下一步 / 继续 / next"这类意图，直接推进流程而不是发给模型。
		if (looksLikeNextIntent(text)) {
			setInput('');
			antMessage.info(t('ai.debate.v2.autoAdvanceHint', {
				defaultValue: 'Detected a "next" intent, advancing to the next step.',
			}));
			onNext();
			return;
		}
		setInput('');
		await onSend(text);
	}

	return (
		<div style={{ display: 'flex', flexDirection: 'column', minHeight: 480 }}>
			<div style={{ marginBottom: 8 }}>
				<Text strong>{stepLabel}</Text>
				<Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
					{t('ai.debate.v2.chatHint', {
						defaultValue: 'Tell me your idea and intent. I will summarize my understanding; when you are happy with it, click Next.',
					})}
				</Text>
			</div>
			<div
				ref={listRef}
				style={{
					flex: 1,
					minHeight: 320,
					maxHeight: 560,
					overflowY: 'auto',
					border: '1px solid #e5e7eb',
					borderRadius: 8,
					padding: 12,
					background: '#fafafa',
				}}
			>
				{state.messages.length === 0 ? (
					<Empty
						description={isFirstChat
							? t('ai.debate.v2.chatEmptyIntent', {
								defaultValue: 'Describe the strategy you want in natural language. The assistant will help you shape it.',
							})
							: t('ai.debate.v2.chatEmptyAgent', {
								defaultValue: 'Chat naturally with the current expert. They will give suggestions and ask questions within their own scope.',
							})}
					/>
				) : (
					state.messages.map((m) => (
						<MessageBubble
							key={m.id}
							m={m}
							waitHint={
								modelWaitActive && m.isLoading
									? t('ai.debate.v2.modelWaitBubble', {
											defaultValue: 'Waiting {{time}}',
											time: formatElapsed(modelWaitElapsedSeconds),
										})
									: undefined
							}
						/>
					))
				)}
				{sending && streamingPreview ? (
					<div style={{ display: 'flex', justifyContent: 'flex-start', marginBottom: 8 }}>
						<div
							style={{
								maxWidth: '85%',
								background: '#ffffff',
								border: '1px solid #e5e7eb',
								borderRadius: 8,
								padding: '8px 12px',
								whiteSpace: 'pre-wrap',
								wordBreak: 'break-word',
							}}
						>
							<Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>
								{t('ai.debate.v2.streamingPreview', { defaultValue: 'Generating…' })}
							</Text>
							<Text>{streamingPreview}</Text>
						</div>
					</div>
				) : null}
			</div>
			{modelWaitActive ? (
				<Alert
					type="info"
					showIcon
					style={{ marginTop: 8 }}
					message={t('ai.debate.v2.modelWaitBanner', {
						defaultValue: 'Model is working… Elapsed {{time}}',
						time: formatElapsed(modelWaitElapsedSeconds),
					})}
				/>
			) : null}
			<div style={{ marginTop: 8, display: 'flex', gap: 8 }}>
				<Input.TextArea
					value={input}
					onChange={(e) => setInput(e.target.value)}
					rows={2}
					placeholder={isFirstChat
						? t('ai.debate.placeholders.intent')
						: t('ai.debate.v2.chatPlaceholder', { defaultValue: 'Say something to the current expert…' })}
					onPressEnter={(e) => {
						if (!e.shiftKey) {
							e.preventDefault();
							void handleSend();
						}
					}}
				/>
				<Button type="primary" loading={sending} onClick={handleSend}>
					{t('ai.debate.v2.send', { defaultValue: 'Send' })}
				</Button>
			</div>

			<Divider style={{ margin: '16px 0 12px' }} />

			<div
				style={{
					display: 'grid',
					gridTemplateColumns: '1fr auto 1fr',
					alignItems: 'center',
					gap: 12,
				}}
			>
				<div>
					<Button onClick={onBack} disabled={!canBack}>
						{t('ai.debate.v2.back', { defaultValue: 'Back' })}
					</Button>
				</div>
				<div style={{ textAlign: 'center' }}>
					<Button
						type="primary"
						size="large"
						onClick={onNext}
						loading={sending}
						disabled={sending}
						title={sending
							? t('ai.debate.v2.nextDisabledWhileSending', {
								defaultValue: 'Please wait until the assistant finishes the current reply.',
							})
							: undefined}
					>
						{isLastAgent
							? t('ai.debate.v2.generateCode', { defaultValue: 'Generate code' })
							: t('ai.debate.v2.next', { defaultValue: 'Next' })}
					</Button>
				</div>
				<div style={{ textAlign: 'right' }}>
					<Text type="secondary" style={{ fontSize: 12 }}>
						{t('ai.debate.v2.nextHint', {
							defaultValue: 'Typing “next” or “continue” also advances the flow.',
						})}
					</Text>
				</div>
			</div>
		</div>
	);
}

function MessageBubble({ m, waitHint }: { m: ChatMessage; waitHint?: string }) {
	// 系统衔接消息仅用于提示 LLM，不展示给最终用户——保持对话干净，
	// 直接由 Agent 的自我介绍开场即可。
	if (m.kind === 'kickoff') {
		return null;
	}
	const isUser = m.role === 'user';
	return (
		<div style={{ display: 'flex', justifyContent: isUser ? 'flex-end' : 'flex-start', marginBottom: 8 }}>
			<div
				style={{
					maxWidth: '85%',
					background: isUser ? '#fff4d6' : '#ffffff',
					border: '1px solid #e5e7eb',
					borderRadius: 8,
					padding: '8px 12px',
					whiteSpace: 'pre-wrap',
					wordBreak: 'break-word',
				}}
			>
				{m.isLoading ? (
					<Space direction="vertical" size={4} align="start">
						<Spin size="small" />
						{waitHint ? (
							<Text type="secondary" style={{ fontSize: 12 }}>
								{waitHint}
							</Text>
						) : null}
					</Space>
				) : (
					<Text>{m.content}</Text>
				)}
			</div>
		</div>
	);
}

// -- Step: Code ---------------------------------------------------------------

export function CodeStep(props: {
	code: { text: string; python: string; loading: boolean };
	onBack: () => void;
	onReject: (feedback: string) => Promise<void>;
	sending: boolean;
	/** Recover from edge timeout / desync: back one step then advance to re-run code generation. */
	onRetryCodeGen?: () => Promise<void>;
}) {
	const { t } = useTranslation();
	const navigate = useNavigate();
	const { code, onBack, onReject, sending, onRetryCodeGen } = props;

	// Reject-to-rewrite modal state.
	const [rejectOpen, setRejectOpen] = useState(false);
	const [rejectText, setRejectText] = useState('');

	// Save-as-template modal state.
	const [saveOpen, setSaveOpen] = useState(false);
	const [saveForm] = Form.useForm<{ name: string; description: string }>();
	const [saving, setSaving] = useState(false);

	const codeToUse = code.python || code.text;
	const hasCode = !!codeToUse && !code.loading;
	const codeSpinTip =
		code.loading && (code.python || code.text)
			? t('ai.debate.v2.codeRegenerating', { defaultValue: 'Rewriting code from your feedback…' })
			: t('ai.debate.v2.codeGenerating', { defaultValue: 'Generating code…' });
	const canAct = hasCode && !sending;

	// 客户端沙箱校验：必须通过才允许保存；未通过时提示"返回重新生成"。
	const violations: Violation[] = useMemo(
		() => (hasCode ? validatePythonSandbox(codeToUse) : []),
		[hasCode, codeToUse],
	);

	// Required-parameter values are NOT collected here — they are filled at
	// backtest/schedule submit time. The save-as-template flow only persists
	// the source code, so saving without param values is safe.
	const isValid = hasCode && violations.length === 0;
	const canSave = canAct && isValid;

	function violationLabel(v: Violation): string {
		// 优先走 i18n key，找不到时用英文 message 作兜底。
		const i18nKey = `ai.debate.v2.validation.codes.${v.code}`;
		const translated = t(i18nKey, { defaultValue: '' });
		const base = translated || v.message;
		return v.hit ? `${base} (${v.hit})` : base;
	}

	function handleAutoRewrite() {
		// 把校验违规当作反馈送回后端重写代码。
		if (violations.length === 0) return;
		Modal.confirm({
			title: t('ai.debate.v2.validation.rewriteConfirmTitle', {
				defaultValue: 'Regenerate the code?',
			}),
			content: t('ai.debate.v2.validation.rewriteConfirmContent', {
				defaultValue: 'The code did not pass the sandbox validator. We will send the violations back to the code agent and regenerate. Continue?',
			}),
			okText: t('ai.debate.v2.validation.rewriteOk', { defaultValue: 'Regenerate' }),
			cancelText: t('ai.debate.v2.validation.rewriteCancel', { defaultValue: 'Cancel' }),
			onOk: async () => {
				await onReject(violationsToFeedback(violations));
			},
		});
	}

	async function handleReject() {
		const fb = rejectText.trim();
		if (!fb) {
			antMessage.warning(t('ai.debate.v2.rejectFeedbackRequired', {
				defaultValue: 'Please describe what to improve so the model can rewrite the code.',
			}));
			return;
		}
		setRejectOpen(false);
		setRejectText('');
		await onReject(fb);
	}

	async function handleSave() {
		let values: { name: string; description: string };
		try {
			values = await saveForm.validateFields();
		} catch {
			return;
		}
		setSaving(true);
		try {
			const resp = await strategyApi.createTemplate({
				name: values.name.trim(),
				description: (values.description || '').trim(),
				code: codeToUse,
				parameters: [],
				isPublic: false,
				tags: ['ai-debate'],
			});
			antMessage.success(t('ai.debate.v2.saveSuccess', { defaultValue: 'Saved as a private template' }));
			setSaveOpen(false);
			saveForm.resetFields();
			// Offer a quick jump to the templates page so the user can run a
			// backtest against the freshly saved template.
			Modal.confirm({
				title: t('ai.debate.v2.saveGotoConfirmTitle', { defaultValue: 'Go to Strategy Templates?' }),
				content: t('ai.debate.v2.saveGotoConfirmContent', {
					defaultValue: 'The template has been saved. Want to open it in Strategy Templates to run a backtest?',
				}),
				okText: t('ai.debate.v2.saveGotoOk', { defaultValue: 'Open templates' }),
				cancelText: t('ai.debate.v2.saveGotoCancel', { defaultValue: 'Stay here' }),
				onOk: () => {
					const id = resp?.id || '';
					if (id) {
						navigate(`/strategy/templates?group=user&templateId=${encodeURIComponent(String(id))}`);
					} else {
						navigate('/strategy/templates?group=user');
					}
				},
			});
		} catch (e: unknown) {
			const msg = e instanceof Error ? e.message : '';
			antMessage.error(msg || t('ai.debate.v2.saveFailed', { defaultValue: 'Failed to save template' }));
		} finally {
			setSaving(false);
		}
	}

	return (
		<div>
			<div style={{ marginBottom: 8 }}>
				<Text strong>{t('ai.debate.v2.codeTitle', { defaultValue: 'Code proposal' })}</Text>
				<Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
					{t('ai.debate.v2.codeHint', { defaultValue: 'Generated from the agreed summaries of all previous steps.' })}
				</Text>
			</div>
			{code.loading ? (
				<>
					<Alert
						type="info"
						showIcon
						style={{ marginBottom: 12 }}
						message={t('ai.debate.v2.modelWaitBanner', {
							defaultValue: 'Model is working… Elapsed {{time}} (duration depends on model and network)',
							time: formatElapsed(code.elapsedSeconds),
						})}
					/>
					{(code.python || code.text) ? (
						<pre
							style={{
								background: '#0f172a',
								color: '#e2e8f0',
								padding: 16,
								borderRadius: 8,
								maxHeight: 360,
								overflow: 'auto',
								fontSize: 13,
								lineHeight: 1.5,
								marginBottom: 16,
								whiteSpace: 'pre-wrap',
								wordBreak: 'break-word',
							}}
						>
							<code>{code.python || code.text}</code>
						</pre>
					) : null}
					<div style={{ textAlign: 'center', padding: code.python || code.text ? 8 : 32 }}>
						<Spin size="large" tip={codeSpinTip} />
						<div style={{ marginTop: 12 }}>
							<Text type="secondary" style={{ fontSize: 12 }}>
								{t('ai.debate.v2.modelWaitBubble', {
									defaultValue: 'Elapsed {{time}}',
									time: formatElapsed(code.elapsedSeconds),
								})}
							</Text>
						</div>
					</div>
				</>
			) : code.python ? (
				<pre
					style={{
						background: '#0f172a',
						color: '#e2e8f0',
						padding: 16,
						borderRadius: 8,
						maxHeight: 520,
						overflow: 'auto',
						fontSize: 13,
						lineHeight: 1.5,
					}}
				>
					<code>{code.python}</code>
				</pre>
			) : code.text ? (
				<Paragraph>
					<pre style={{ whiteSpace: 'pre-wrap' }}>{code.text}</pre>
				</Paragraph>
			) : (
				<div>
					{onRetryCodeGen ? (
						<Alert
							type="warning"
							showIcon
							style={{ marginBottom: 12 }}
							message={t('ai.debate.v2.codeMissingHint', {
								defaultValue:
									'If the gateway timed out (e.g. HTTP 524), the session may already be on this step without code. Use the button below to go back one step and trigger code generation again.',
							})}
							action={
								<Button type="primary" loading={sending} onClick={() => void onRetryCodeGen()}>
									{t('ai.debate.v2.retryCodeGen', { defaultValue: 'Try generating code again' })}
								</Button>
							}
						/>
					) : null}
					<Empty description={t('ai.debate.v2.codeEmpty', { defaultValue: 'Code not generated yet.' })} />
				</div>
			)}

			{hasCode && isValid && (
				<div style={{ marginTop: 12 }}>
					<CodeExplainPanel code={codeToUse} />
				</div>
			)}

			{hasCode && (
				<div style={{ marginTop: 12 }}>
					{isValid ? (
						(() => {
							const passDesc = t('ai.debate.v2.validation.passDesc', { defaultValue: '' });
							return (
								<Alert
									type="success"
									showIcon
									message={t('ai.debate.v2.validation.passTitle', {
										defaultValue: 'Code validation passed. You can save it as a template.',
									})}
									description={passDesc || undefined}
								/>
							);
						})()
					) : (
						<Alert
							type="error"
							showIcon
							message={t('ai.debate.v2.validation.failTitle', {
								defaultValue: 'Sandbox validation failed',
							})}
							description={(
								<div>
									<div style={{ marginBottom: 6 }}>
										{t('ai.debate.v2.validation.failDesc', {
											defaultValue: 'The following issues block saving. You can ask the code agent to regenerate.',
										})}
									</div>
									<ul style={{ margin: 0, paddingLeft: 20 }}>
										{violations.map((v, i) => (
											<li key={`${v.code}-${i}`}>{violationLabel(v)}</li>
										))}
									</ul>
									<div style={{ marginTop: 8 }}>
										<Button
											type="primary"
											danger
											size="small"
											loading={sending}
											onClick={handleAutoRewrite}
										>
											{t('ai.debate.v2.validation.rewriteBtn', {
												defaultValue: 'Send violations back & regenerate',
											})}
										</Button>
									</div>
								</div>
							)}
						/>
					)}
				</div>
			)}

			<div style={{ marginTop: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Button onClick={onBack}>{t('ai.debate.v2.back', { defaultValue: 'Back' })}</Button>
				<Space>
					<Button
						danger
						disabled={!canAct}
						onClick={() => setRejectOpen(true)}
					>
						{t('ai.debate.v2.rejectCode', { defaultValue: 'Reject & rewrite' })}
					</Button>
					<Button
						type="primary"
						disabled={!canSave}
						title={!canSave && hasCode
							? t('ai.debate.v2.validation.saveBlocked', {
								defaultValue: 'Save is disabled until the code passes sandbox validation.',
							})
							: undefined}
						onClick={() => setSaveOpen(true)}
					>
						{t('ai.debate.v2.saveTemplate', { defaultValue: 'Save as template' })}
					</Button>
				</Space>
			</div>

			<Modal
				title={t('ai.debate.v2.rejectModalTitle', { defaultValue: 'Reject current code & rewrite' })}
				open={rejectOpen}
				onOk={handleReject}
				onCancel={() => setRejectOpen(false)}
				okText={t('ai.debate.v2.rejectModalOk', { defaultValue: 'Regenerate' })}
				cancelText={t('ai.debate.v2.rejectModalCancel', { defaultValue: 'Cancel' })}
				confirmLoading={sending}
				destroyOnClose
			>
				<Paragraph type="secondary" style={{ marginTop: 0 }}>
					{t('ai.debate.v2.rejectModalHint', {
						defaultValue: 'Describe what to fix (e.g. use RSI instead of MACD, tighten stop-loss to 0.5%, avoid trading on news days...). The model will regenerate the code based on your feedback.',
					})}
				</Paragraph>
				<Input.TextArea
					rows={5}
					value={rejectText}
					onChange={(e) => setRejectText(e.target.value)}
					placeholder={t('ai.debate.v2.rejectModalPlaceholder', {
						defaultValue: 'What should be changed?',
					})}
				/>
			</Modal>

			<Modal
				title={t('ai.debate.v2.saveModalTitle', { defaultValue: 'Save as strategy template' })}
				open={saveOpen}
				onOk={handleSave}
				onCancel={() => setSaveOpen(false)}
				okText={t('ai.debate.v2.saveModalOk', { defaultValue: 'Save' })}
				cancelText={t('ai.debate.v2.saveModalCancel', { defaultValue: 'Cancel' })}
				confirmLoading={saving}
				destroyOnClose
			>
				<Form form={saveForm} layout="vertical">
					<Form.Item
						name="name"
						label={t('ai.debate.v2.saveFieldName', { defaultValue: 'Template name' })}
						rules={[{ required: true, message: t('ai.debate.v2.saveNameRequired', { defaultValue: 'Name is required' }) }]}
					>
						<Input maxLength={80} />
					</Form.Item>
					<Form.Item
						name="description"
						label={t('ai.debate.v2.saveFieldDesc', { defaultValue: 'Description (optional)' })}
					>
						<Input.TextArea rows={3} maxLength={500} />
					</Form.Item>
				</Form>
			</Modal>
		</div>
	);
}
