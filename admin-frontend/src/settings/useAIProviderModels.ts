import { useRequest } from 'ahooks';
import { useMemo } from 'react';
import type { AIProvider } from './AIProviderSettings';
import { openchatAPIBase, openchatRequest } from './openchat-api';

interface Options {
	robotCode: string;
	scope: 'chat_room' | 'friend';
	targetId: string;
}

export const useAIProviderModels = ({ robotCode, scope, targetId }: Options) => {
	const apiURL = `${openchatAPIBase(robotCode)}/ai-providers?scope=${scope}&target_id=${encodeURIComponent(targetId)}`;
	const { data: providers = [], loading } = useRequest(() => openchatRequest<AIProvider[]>(apiURL), {
		refreshDeps: [apiURL],
	});
	const selectedProvider =
		providers.find(provider => provider.target_selected) || providers.find(provider => provider.global_selected);
	const modelOptions = useMemo(
		() => (selectedProvider?.available_models || []).map(model => ({ value: model })),
		[selectedProvider],
	);
	return { loading, modelOptions, providers, selectedProvider };
};
