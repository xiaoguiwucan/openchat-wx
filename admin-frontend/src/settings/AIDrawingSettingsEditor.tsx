import Editor from '@monaco-editor/react';
import { AutoComplete, Button, Select, Space, Typography } from 'antd';
import React from 'react';
import { registerMonacoJsonSchema } from './monacoJsonSchema';
import { defaultAIDrawingValue } from './utils';

interface IProps {
	value?: string;
	onChange?: (value?: string) => void;
	modelOptions?: Array<{ value: string }>;
	channelManaged?: boolean;
}

const parseDrawingSettings = (value?: string): Record<string, unknown> => {
	if (!value) return {};
	try {
		const parsed = JSON.parse(value) as unknown;
		return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? (parsed as Record<string, unknown>) : {};
	} catch {
		return {};
	}
};

const getDrawingModel = (settings: Record<string, unknown>) => {
	if (typeof settings.model === 'string') return settings.model;
	for (const key of ['openai_compatible', 'openai-compatible', 'custom', 'OpenAI']) {
		const candidate = settings[key];
		if (candidate && typeof candidate === 'object' && !Array.isArray(candidate)) {
			const model = (candidate as Record<string, unknown>).model;
			if (typeof model === 'string') return model;
		}
	}
	return '';
};

const AIDrawingSettingsEditor = (props: IProps) => {
	const settings = parseDrawingSettings(props.value);
	const drawingModel = getDrawingModel(settings);
	const updateChannelParameter = (key: 'size' | 'quality' | 'response_format', value?: string) => {
		const next = {
			size: typeof settings.size === 'string' ? settings.size : undefined,
			quality: typeof settings.quality === 'string' ? settings.quality : undefined,
			response_format: typeof settings.response_format === 'string' ? settings.response_format : undefined,
			[key]: value,
		};
		props.onChange?.(JSON.stringify(Object.fromEntries(Object.entries(next).filter(([, item]) => item)), null, 2));
	};
	const updateDrawingModel = (model: string) => {
		props.onChange?.(JSON.stringify({ ...parseDrawingSettings(props.value), model }, null, 2));
	};

	if (props.channelManaged) {
		return (
			<Space
				orientation="vertical"
				size={12}
				style={{ width: '100%' }}
			>
				<div>
					<Typography.Text>图片尺寸</Typography.Text>
					<Select
						value={typeof settings.size === 'string' ? settings.size : undefined}
						placeholder="留空则使用渠道默认值"
						allowClear
						style={{ width: '100%', marginTop: 6 }}
						options={['1024x1024', '1536x1024', '1024x1536', '2048x2048', '2048x1152', '3840x2160', '2160x3840'].map(
							value => ({ value }),
						)}
						onChange={value => updateChannelParameter('size', value)}
					/>
				</div>
				<div>
					<Typography.Text>图片质量</Typography.Text>
					<Select
						value={typeof settings.quality === 'string' ? settings.quality : undefined}
						placeholder="留空则使用渠道默认值"
						allowClear
						style={{ width: '100%', marginTop: 6 }}
						options={['auto', 'low', 'medium', 'high'].map(value => ({ value }))}
						onChange={value => updateChannelParameter('quality', value)}
					/>
				</div>
				<div>
					<Typography.Text>返回格式</Typography.Text>
					<Select
						value={typeof settings.response_format === 'string' ? settings.response_format : undefined}
						placeholder="留空则自动选择"
						allowClear
						style={{ width: '100%', marginTop: 6 }}
						options={[
							{ label: '自动', value: 'auto' },
							{ label: '链接', value: 'url' },
							{ label: 'Base64', value: 'b64_json' },
						]}
						onChange={value => updateChannelParameter('response_format', value)}
					/>
				</div>
			</Space>
		);
	}

	return (
		<div>
			<Space
				orientation="vertical"
				size={6}
				style={{ width: '100%', marginBottom: 12 }}
			>
				<Typography.Text>生图模型</Typography.Text>
				<AutoComplete
					value={drawingModel}
					options={props.modelOptions}
					placeholder="从当前渠道选择或手动输入生图模型"
					style={{ width: '100%' }}
					showSearch={{ filterOption: () => true }}
					onChange={updateDrawingModel}
				/>
			</Space>
			<div
				style={{
					position: 'relative',
					border: '1px solid #d9d9d9',
					borderRadius: 6,
					padding: '8px 2px',
				}}
			>
				<Editor
					width="100%"
					height="450px"
					language="json"
					options={{
						minimap: { enabled: false },
						scrollBeyondLastLine: false,
						tabSize: 2,
						insertSpaces: true,
						fixedOverflowWidgets: true,
						scrollbar: { alwaysConsumeMouseWheel: false },
					}}
					value={props.value}
					onChange={props.onChange}
					onMount={(editor, monaco) => {
						const model = editor.getModel();
						if (model) {
							registerMonacoJsonSchema(monaco, model.uri.toString(), 'http://myserver/ai-drawing-schema.json', {
								type: 'object',
								properties: {
									'Z-Image': {
										type: 'object',
										properties: {
											enabled: {
												type: 'boolean',
												description: '是否启用',
											},
											base_url: {
												type: 'string',
											},
											model: {
												type: 'string',
											},
											api_key: {
												type: 'string',
											},
										},
										required: ['enabled', 'base_url', 'model', 'api_key'],
										description: '造像绘图',
									},
									GLM: {
										type: 'object',
										properties: {
											enabled: {
												type: 'boolean',
												description: '是否启用',
											},
										},
										required: ['enabled'],
										description: '智谱',
									},
									JiMeng: {
										type: 'object',
										properties: {
											base_url: {
												type: 'string',
											},
											model: {
												type: 'string',
											},
											sessionid: {
												type: 'array',
												items: {
													type: 'string',
												},
											},
											sample_strength: {
												type: 'number',
											},
											resolution: {
												type: 'string',
											},
											ratio: {
												type: 'string',
											},
											response_format: {
												type: 'string',
											},
											enabled: {
												type: 'boolean',
												description: '是否启用',
											},
										},
										required: ['base_url', 'model', 'sessionid', 'enabled'],
										description: '即梦绘图',
									},
									DouBao: {
										type: 'object',
										properties: {
											enabled: {
												type: 'boolean',
												description: '是否启用',
											},
											api_key: {
												type: 'string',
											},
											model: {
												type: 'string',
											},
											response_format: {
												type: 'string',
											},
											watermark: {
												type: 'boolean',
												description: '是否包含水印',
											},
											size: {
												type: 'string',
											},
										},
										required: ['enabled', 'api_key', 'model'],
										description: '豆包绘图',
									},
									OpenAI: {
										type: 'object',
										properties: {
											n: {
												type: 'integer',
												description: '一次生成多少张图片',
											},
											size: {
												type: 'string',
												enum: [
													'auto',
													'1024x1024',
													'1536x1024',
													'1024x1536',
													'2048x2048',
													'2048x1152',
													'3840x2160',
													'2160x3840',
												],
											},
											quality: {
												type: 'string',
												enum: ['auto', 'low', 'medium', 'high'],
											},
											background: {
												type: 'string',
												enum: ['auto', 'opaque'],
											},
											output_format: {
												type: 'string',
												enum: ['png', 'jpeg', 'webp'],
											},
										},
										required: ['n', 'size', 'quality', 'background', 'output_format'],
										description: 'OpenAI 绘图',
									},
								},
								required: ['JiMeng', 'DouBao', 'GLM', 'Z-Image', 'OpenAI'],
							});
						}
					}}
				/>
				<div style={{ position: 'absolute', top: 4, right: 4, zIndex: 9999 }}>
					<Button
						color="default"
						variant="filled"
						onClick={() => {
							props.onChange?.(defaultAIDrawingValue);
						}}
					>
						重置为默认值
					</Button>
				</div>
			</div>
		</div>
	);
};

export default React.memo(AIDrawingSettingsEditor);
