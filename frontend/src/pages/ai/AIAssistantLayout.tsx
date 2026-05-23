import { useEffect } from 'react';
import { Tabs } from 'antd';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAgentStore } from './agentStore';

export default function AIAssistantLayout() {
	const { t } = useTranslation();
	const location = useLocation();
	const navigate = useNavigate();
	const preloadAgents = useAgentStore((s) => s.preload);

	// 进入任意 /ai/* 页面时，立即在布局级别预加载一次 Agent 定义。
	// `preload()` 内部做了幂等保护，不会造成重复请求，这样 /ai/debate 与
	// /ai/agents 之间相互切换时可以瞬间复用数据。
	useEffect(() => {
		void preloadAgents();
	}, [preloadAgents]);

	const tabItems = [
		{ key: '/ai/debate', label: t('ai.tabs.debate', { defaultValue: t('ai.debate.title') }) },
		{ key: '/ai/settings', label: t('ai.tabs.settings', { defaultValue: t('ai.settings.pageTitle') }) },
		{ key: '/ai/agents', label: t('ai.tabs.agentSettings', { defaultValue: t('ai.settings.agent.title') }) },
	];

	const activeKey = (() => {
		const p = location.pathname || '';
		if (p.startsWith('/ai/debate')) return '/ai/debate';
		if (p.startsWith('/ai/agents')) return '/ai/agents';
		if (p.startsWith('/ai/settings')) return '/ai/settings';
		return '/ai/debate';
	})();

	return (
		<div className="ai-assistant-scope">
			<Tabs
				activeKey={activeKey}
				items={tabItems}
				onChange={(key) => navigate(key)}
			/>
			<Outlet />
		</div>
	);
}
