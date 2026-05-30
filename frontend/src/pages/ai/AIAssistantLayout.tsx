import { useEffect } from 'react';
import { Tabs } from 'antd';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/queries/queryKeys';
import { aiApi } from '@/client/ai';

export default function AIAssistantLayout() {
	const { t } = useTranslation();
	const location = useLocation();
	const navigate = useNavigate();
	const queryClient = useQueryClient();

	// Eagerly prefetch agent definitions so /ai/debate and /ai/agents
	// have instant data when navigating between tabs.
	useEffect(() => {
		queryClient.prefetchQuery({
			queryKey: queryKeys.ai.agents.list(),
			queryFn: () => aiApi.listAgents(),
			staleTime: 5 * 60_000,
		});
	}, [queryClient]);

	const tabItems = [
		{ key: '/ai/debate', label: t('ai.tabs.debate', { defaultValue: t('ai.debate.title') }) },
		{ key: '/ai/settings', label: t('ai.tabs.settings', { defaultValue: t('ai.settings.pageTitle') }) },
		{ key: '/ai/agents', label: t('ai.tabs.agentSettings', { defaultValue: t('ai.settings.agent.title') }) },
		{ key: '/ai/gate', label: t('ai.tabs.gate', { defaultValue: 'AI Gate' }) },
	];

	const activeKey = (() => {
		const p = location.pathname || '';
		if (p.startsWith('/ai/debate')) return '/ai/debate';
		if (p.startsWith('/ai/agents')) return '/ai/agents';
		if (p.startsWith('/ai/settings')) return '/ai/settings';
		if (p.startsWith('/ai/gate')) return '/ai/gate';
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
