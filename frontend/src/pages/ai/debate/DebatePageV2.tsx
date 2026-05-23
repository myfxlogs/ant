import { useEffect } from 'react';
import { Alert, Button, Card, Space, Steps, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import { useAgentStore } from '../agentStore';
import { useDebateFlow } from './flow/useDebateFlow';
import { AgentSelectionStep, ChatStep, CodeStep } from './DebatePageV2Steps';

const { Text } = Typography;

/**
 * Phase 1 skeleton of the redesigned debate flow:
 *
 *   Step 1 - 选 Agent
 *   Step 2 - 意图澄清对话
 *   Step 3..N - 按顺序与每个 Agent 对话
 *   Step N+1 - 代码生成
 *
 * 所有状态都在前端内存中，LLM 调用走现成的 aiApi.chat。
 * 后端 DebateSession 持久化与真正的 Agent 身份注入会在 phase 2 补齐。
 */
export default function DebatePageV2() {
	const { t } = useTranslation();
	const agentDefs = useAgentStore((s) => s.agentDefs);
	const agentsLoading = useAgentStore((s) => s.loading);
	const preloadAgents = useAgentStore((s) => s.preload);

	useEffect(() => {
		void preloadAgents();
	}, [preloadAgents]);

	const flow = useDebateFlow();
	const {
		currentStep, stepIndex, stepLabels,
		selectedAgents, setSelectedAgents,
		stepState,
		sending, modelWaitActive, modelWaitElapsedSeconds,
		sendMessage, startFlow, advance, back, reset,
		rejectCode, retryCodeGeneration,
		code,
		advanceStreamPreview,
		provider, model, usage,
	} = flow;

	const showModelBanner = currentStep !== 'agent_selection' && (model || usage.totalTokens > 0);

	return (
		<div style={{ padding: 16 }}>
			<Card
				title={t('ai.debate.title', { defaultValue: '多智能体讨论' })}
				extra={(
					<Space>
						<Button size="small" onClick={reset}>
							{t('ai.debate.v2.reset', { defaultValue: '重置' })}
						</Button>
					</Space>
				)}
			>
				{showModelBanner ? (
					<Alert
						className="ai-gold-alert"
						type="info"
						showIcon
						style={{ marginBottom: 12 }}
						message={(
							<Space size="middle" wrap>
								<span>
									<Text type="secondary">{t('ai.debate.v2.currentModel', { defaultValue: '当前模型' })}：</Text>
									<Text strong>{model || t('ai.debate.v2.modelUnknown', { defaultValue: '未配置' })}</Text>
									{provider ? <Text type="secondary"> · {provider}</Text> : null}
								</span>
								<Text type="secondary">
									{t('ai.debate.v2.tokenUsage', {
										defaultValue: 'Tokens: prompt {{p}} / completion {{c}} / total {{n}}',
										p: usage.promptTokens,
										c: usage.completionTokens,
										n: usage.totalTokens,
									})}
								</Text>
							</Space>
						)}
					/>
				) : null}
				<Steps
					size="small"
					current={stepIndex}
					items={stepLabels.map((s) => ({ title: s.label }))}
					style={{ marginBottom: 16 }}
				/>

				{currentStep === 'agent_selection' ? (
					<AgentSelectionStep
						agentDefs={agentDefs}
						agentsLoading={agentsLoading}
						selectedAgents={selectedAgents}
						onChange={setSelectedAgents}
						onNext={startFlow}
					/>
				) : currentStep === 'code' ? (
					<CodeStep
						code={code}
						onBack={back}
						onReject={rejectCode}
						sending={sending}
						onRetryCodeGen={retryCodeGeneration}
					/>
				) : (
					<ChatStep
						stepKey={currentStep}
						stepLabel={stepLabels[stepIndex]?.label || ''}
						state={stepState(currentStep)}
						sending={sending}
						modelWaitActive={modelWaitActive}
						modelWaitElapsedSeconds={modelWaitElapsedSeconds}
						streamingPreview={advanceStreamPreview}
						onSend={sendMessage}
						onBack={back}
						onNext={advance}
						isFirstChat={stepIndex === 1}
						isLastAgent={currentStep !== 'intent' && stepIndex === stepLabels.length - 2}
						canBack={currentStep !== 'intent'}
					/>
				)}
			</Card>
		</div>
	);
}
