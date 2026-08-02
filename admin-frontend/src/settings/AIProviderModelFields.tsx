import { Form, Select, Space } from 'antd';
import type { FormInstance } from 'antd';
import type { Rule } from 'antd/es/form';
import type { ReactNode } from 'react';
import { useMemo } from 'react';
import type { AIProvider } from './AIProviderSettings';

type ProviderModelKey = 'chat_model' | 'image_recognition_model' | 'image_generation_model' | 'summary_model';

interface Props {
	form: FormInstance;
	providers: AIProvider[];
	providerName: string;
	modelName: string;
	providerLabel: string;
	modelLabel: string;
	defaultModelKey?: ProviderModelKey;
	modelKind?: 'all' | 'embedding';
	required?: boolean;
	allowInherit?: boolean;
	modelHelp?: ReactNode;
	modelTooltip?: ReactNode;
}

const providerModels = (provider: AIProvider | undefined) => {
	if (!provider) return [];
	return Array.from(
		new Set(
			[
				...(provider.available_models || []),
				provider.chat_model,
				provider.image_recognition_model,
				provider.image_generation_model,
				provider.summary_model,
			].filter(Boolean),
		),
	);
};

const AIProviderModelFields = ({
	form,
	providers,
	providerName,
	modelName,
	providerLabel,
	modelLabel,
	defaultModelKey,
	modelKind = 'all',
	required = true,
	allowInherit = false,
	modelHelp,
	modelTooltip,
}: Props) => {
	const providerID = Form.useWatch(providerName, form) as number | undefined;
	const enabledProviders = useMemo(() => providers.filter(provider => provider.enabled), [providers]);
	const selectedProvider = enabledProviders.find(provider => provider.id === providerID);
	const modelOptions = useMemo(
		() =>
			providerModels(selectedProvider)
				.filter(model => modelKind !== 'embedding' || /(embedding|embed|bge|gte|m3e|e5)/i.test(model))
				.map(model => ({ label: model, value: model })),
		[modelKind, selectedProvider],
	);
	const optionalPairRules: Rule[] | undefined = required
		? undefined
		: [
				({ getFieldValue }) => ({
					validator: (_: unknown, value: unknown) =>
						getFieldValue(providerName) && !value
							? Promise.reject(new Error(`选择${providerLabel}后必须选择${modelLabel}`))
							: Promise.resolve(),
				}),
			];

	return (
		<Space
			align="start"
			size={12}
			wrap
			style={{ width: '100%' }}
		>
			<Form.Item
				name={providerName}
				label={providerLabel}
				rules={required ? [{ required: true, message: `${providerLabel}不能为空` }] : undefined}
				style={{ flex: '1 1 240px', marginBottom: 16 }}
			>
				<Select
					showSearch={{ optionFilterProp: 'label' }}
					placeholder={required ? `请选择${providerLabel}` : `请选择${providerLabel}`}
					options={[
						...(required ? [] : [{ label: allowInherit ? '继承全局设置' : '不使用模型渠道', value: 0 }]),
						...enabledProviders.map(provider => ({ label: provider.name, value: provider.id })),
					]}
					onChange={(nextProviderID: number) => {
						const provider = enabledProviders.find(item => item.id === nextProviderID);
						const models = providerModels(provider);
						form.setFieldValue(modelName, defaultModelKey ? provider?.[defaultModelKey] || models[0] || '' : '');
					}}
				/>
			</Form.Item>
			<Form.Item
				name={modelName}
				label={modelLabel}
				dependencies={[providerName]}
				rules={required ? [{ required: true, message: `${modelLabel}不能为空` }] : optionalPairRules}
				help={modelHelp}
				tooltip={modelTooltip}
				style={{ flex: '2 1 360px', marginBottom: 16 }}
			>
				<Select
					placeholder={
						selectedProvider
							? `从“${selectedProvider.name}”选择或输入模型`
							: required
								? `请先选择${providerLabel}`
								: allowInherit
									? '继承全局模型'
									: '未选择模型渠道'
					}
					options={modelOptions}
					disabled={!selectedProvider}
					showSearch={{ optionFilterProp: 'label' }}
					allowClear={!required}
				/>
			</Form.Item>
		</Space>
	);
};

export default AIProviderModelFields;
