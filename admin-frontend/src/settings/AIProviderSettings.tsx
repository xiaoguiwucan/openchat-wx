import {
	ApiOutlined,
	CheckCircleFilled,
	DeleteOutlined,
	EditOutlined,
	PlusOutlined,
	ThunderboltOutlined,
} from '@ant-design/icons';
import { App, Button, Form, Input, Modal, Popconfirm, Space, Switch, Table, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import ParamsGroup from '@/components/ParamsGroup';
import { openchatAPIBase, openchatRequest } from './openchat-api';

interface IProps {
	robotCode: string;
}

interface AIProvider {
	id: number;
	name: string;
	base_url: string;
	api_key_masked: string;
	has_api_key: boolean;
	chat_model: string;
	image_recognition_model: string;
	image_generation_model: string;
	summary_model: string;
	image_size: string;
	image_quality: string;
	enabled: boolean;
	global_selected: boolean;
}

interface AIProviderForm {
	name: string;
	base_url: string;
	api_key?: string;
	chat_model: string;
	image_recognition_model?: string;
	image_generation_model?: string;
	summary_model?: string;
	image_size?: string;
	image_quality?: string;
	enabled: boolean;
}

const AIProviderSettings = ({ robotCode }: IProps) => {
	const { message } = App.useApp();
	const [form] = Form.useForm<AIProviderForm>();
	const [providers, setProviders] = useState<AIProvider[]>([]);
	const [loading, setLoading] = useState(false);
	const [saving, setSaving] = useState(false);
	const [testingId, setTestingId] = useState<number>();
	const [editing, setEditing] = useState<AIProvider>();
	const [modalOpen, setModalOpen] = useState(false);
	const apiBase = useMemo(() => `${openchatAPIBase(robotCode)}/ai-providers`, [robotCode]);

	const loadProviders = useCallback(async () => {
		setLoading(true);
		try {
			setProviders(await openchatRequest<AIProvider[]>(`${apiBase}?scope=global`));
		} catch (error) {
			message.error(error instanceof Error ? error.message : '模型渠道读取失败');
		} finally {
			setLoading(false);
		}
	}, [apiBase, message]);

	useEffect(() => {
		void loadProviders();
	}, [loadProviders]);

	const openEditor = (provider?: AIProvider) => {
		setEditing(provider);
		form.resetFields();
		form.setFieldsValue(
			provider
				? {
						...provider,
						api_key: undefined,
				  }
				: { enabled: true, image_size: '1024x1024', image_quality: '' },
		);
		setModalOpen(true);
	};

	const saveProvider = async () => {
		const values = await form.validateFields();
		setSaving(true);
		try {
			await openchatRequest<AIProvider>(editing ? `${apiBase}/${editing.id}` : apiBase, {
				method: editing ? 'PUT' : 'POST',
				body: JSON.stringify(values),
			});
			message.success(editing ? '模型渠道已更新' : '模型渠道已创建');
			setModalOpen(false);
			form.resetFields();
			await loadProviders();
		} catch (error) {
			message.error(error instanceof Error ? error.message : '模型渠道保存失败');
		} finally {
			setSaving(false);
		}
	};

	const selectProvider = async (provider: AIProvider) => {
		try {
			await openchatRequest<null>(`${apiBase}/select`, {
				method: 'POST',
				body: JSON.stringify({ provider_id: provider.id, scope: 'global', target_id: '' }),
			});
			message.success(`已将“${provider.name}”设为全局默认渠道`);
			await loadProviders();
		} catch (error) {
			message.error(error instanceof Error ? error.message : '切换渠道失败');
		}
	};

	const testProvider = async (provider: AIProvider) => {
		setTestingId(provider.id);
		try {
			const result = await openchatRequest<{ success: boolean; latency_ms: number; message: string }>(
				`${apiBase}/${provider.id}/test`,
				{ method: 'POST', body: '{}' },
			);
			if (result.success) {
				message.success(`${result.message}，耗时 ${result.latency_ms}ms`);
			} else {
				message.error(result.message);
			}
		} catch (error) {
			message.error(error instanceof Error ? error.message : '连接测试失败');
		} finally {
			setTestingId(undefined);
		}
	};

	const deleteProvider = async (provider: AIProvider) => {
		try {
			await openchatRequest<null>(`${apiBase}/${provider.id}`, { method: 'DELETE' });
			message.success('模型渠道已删除');
			await loadProviders();
		} catch (error) {
			message.error(error instanceof Error ? error.message : '删除渠道失败');
		}
	};

	const columns: ColumnsType<AIProvider> = [
		{
			title: '渠道',
			dataIndex: 'name',
			width: 190,
			render: (_, provider) => (
				<Space
					orientation="vertical"
					size={2}
				>
					<Space size={6}>
						<Typography.Text strong>{provider.name}</Typography.Text>
						{provider.global_selected && <Tag color="success">全局默认</Tag>}
						{!provider.enabled && <Tag>已停用</Tag>}
					</Space>
					<Typography.Text
						type="secondary"
						ellipsis={{ tooltip: provider.base_url }}
						style={{ maxWidth: 260 }}
					>
						{provider.base_url}
					</Typography.Text>
				</Space>
			),
		},
		{
			title: '能力模型',
			key: 'models',
			render: (_, provider) => (
				<Space
					orientation="vertical"
					size={1}
				>
					<Typography.Text>对话：{provider.chat_model}</Typography.Text>
					<Typography.Text type="secondary">
						识图：{provider.image_recognition_model || '未配置'} · 生图：{provider.image_generation_model || '未配置'} ·
						 总结：{provider.summary_model || provider.chat_model}
					</Typography.Text>
				</Space>
			),
		},
		{
			title: '操作',
			key: 'actions',
			width: 180,
			align: 'right',
			render: (_, provider) => (
				<Space size={2}>
					<Tooltip title="测试对话模型连接">
						<Button
							type="text"
							icon={<ThunderboltOutlined />}
							loading={testingId === provider.id}
							onClick={() => void testProvider(provider)}
						/>
					</Tooltip>
					{!provider.global_selected && (
						<Tooltip title="设为全局默认">
							<Button
								type="text"
								icon={<CheckCircleFilled />}
								disabled={!provider.enabled}
								onClick={() => void selectProvider(provider)}
							/>
						</Tooltip>
					)}
					<Tooltip title="编辑渠道">
						<Button
							type="text"
							icon={<EditOutlined />}
							onClick={() => openEditor(provider)}
						/>
					</Tooltip>
					<Popconfirm
						title="删除模型渠道"
						description="正在使用的渠道需要先切换后才能删除。"
						onConfirm={() => void deleteProvider(provider)}
					>
						<Tooltip title="删除渠道">
							<Button
								type="text"
								danger
								icon={<DeleteOutlined />}
							/>
						</Tooltip>
					</Popconfirm>
				</Space>
			),
		},
	];

	return (
		<ParamsGroup
			title={
				<Space size={6}>
					<ApiOutlined />
					模型渠道
				</Space>
			}
			style={{ marginTop: 10, paddingBottom: 14 }}
		>
			<Space
				style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 12 }}
				wrap
			>
				<Typography.Text type="secondary">
					每个渠道可分别指定 AI 回复、识图、生图和群聊总结模型。
				</Typography.Text>
				<Button
					type="primary"
					icon={<PlusOutlined />}
					onClick={() => openEditor()}
				>
					新增渠道
				</Button>
			</Space>
			<Table
				rowKey="id"
				size="small"
				loading={loading}
				columns={columns}
				dataSource={providers}
				pagination={false}
				scroll={{ x: 760 }}
				locale={{ emptyText: '尚未配置模型渠道' }}
			/>
			<Modal
				title={editing ? '编辑模型渠道' : '新增模型渠道'}
				open={modalOpen}
				onCancel={() => setModalOpen(false)}
				onOk={() => void saveProvider()}
				confirmLoading={saving}
				okText="保存"
				destroyOnHidden
			>
				<Form
					form={form}
					layout="vertical"
					preserve={false}
					initialValues={{ enabled: true, image_size: '1024x1024' }}
				>
					<Form.Item
						name="enabled"
						label="启用渠道"
						valuePropName="checked"
					>
						<Switch />
					</Form.Item>
					<Form.Item
						name="name"
						label="渠道名称"
						rules={[{ required: true, message: '请输入渠道名称' }]}
					>
						<Input placeholder="例如 Bigsea 主渠道" />
					</Form.Item>
					<Form.Item
						name="base_url"
						label="Base URL"
						rules={[
							{ required: true, message: '请输入 Base URL' },
							{ pattern: /^https?:\/\//, message: 'Base URL 必须以 http:// 或 https:// 开头' },
						]}
					>
						<Input placeholder="https://example.com/v1" />
					</Form.Item>
					<Form.Item
						name="api_key"
						label="API Key"
						rules={[{ required: !editing, message: '请输入 API Key' }]}
						extra={editing ? `当前密钥：${editing.api_key_masked}，留空表示不修改` : undefined}
					>
						<Input.Password
							autoComplete="new-password"
							placeholder={editing ? '留空表示保持当前密钥' : '请输入 API Key'}
						/>
					</Form.Item>
					<Form.Item
						name="chat_model"
						label="AI 回复模型"
						rules={[{ required: true, message: '请输入 AI 回复模型' }]}
					>
						<Input placeholder="例如 gpt-5.6-luna" />
					</Form.Item>
					<Form.Item
						name="image_recognition_model"
						label="识图模型"
					>
						<Input placeholder="留空时使用 AI 回复模型" />
					</Form.Item>
					<Form.Item
						name="image_generation_model"
						label="生图模型"
					>
						<Input placeholder="例如 gpt-image-2" />
					</Form.Item>
					<Form.Item
						name="summary_model"
						label="群聊总结模型"
					>
						<Input placeholder="留空时使用 AI 回复模型" />
					</Form.Item>
					<Space.Compact block>
						<Form.Item
							name="image_size"
							label="生图尺寸"
							style={{ width: '50%' }}
						>
							<Input placeholder="1024x1024" />
						</Form.Item>
						<Form.Item
							name="image_quality"
							label="生图质量"
							style={{ width: '50%' }}
						>
							<Input placeholder="可留空" />
						</Form.Item>
					</Space.Compact>
				</Form>
			</Modal>
		</ParamsGroup>
	);
};

export default AIProviderSettings;
