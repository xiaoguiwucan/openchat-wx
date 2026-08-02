import { CommentOutlined, SaveOutlined } from '@ant-design/icons';
import { App, Button, Form, InputNumber, Select, Space, Switch, Typography } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import ParamsGroup from '@/components/ParamsGroup';
import { openchatAPIBase, openchatRequest } from './openchat-api';

interface IProps {
	robotCode: string;
}

interface FreeReplyForm {
	free_reply_enabled: boolean;
	free_reply_level: 'active' | 'normal' | 'cautious';
	free_reply_cooldown_seconds: number;
	free_reply_daily_limit: number;
}

const FreeReplySettings = ({ robotCode }: IProps) => {
	const { message } = App.useApp();
	const [form] = Form.useForm<FreeReplyForm>();
	const [loading, setLoading] = useState(false);
	const apiURL = useMemo(() => `${openchatAPIBase(robotCode)}/free-reply-settings`, [robotCode]);

	const load = useCallback(async () => {
		setLoading(true);
		try {
			form.setFieldsValue(await openchatRequest<FreeReplyForm>(apiURL));
		} catch (error) {
			message.error(error instanceof Error ? error.message : '自由回复设置读取失败');
		} finally {
			setLoading(false);
		}
	}, [apiURL, form, message]);

	useEffect(() => {
		void load();
	}, [load]);

	const save = async () => {
		const values = await form.validateFields();
		setLoading(true);
		try {
			await openchatRequest<null>(apiURL, { method: 'POST', body: JSON.stringify(values) });
			message.success('自由回复设置已保存');
			await load();
		} catch (error) {
			message.error(error instanceof Error ? error.message : '自由回复设置保存失败');
		} finally {
			setLoading(false);
		}
	};

	return (
		<ParamsGroup
			title={
				<Space size={6}>
					<CommentOutlined />
					自由回复
				</Space>
			}
			style={{ marginTop: 24, paddingBottom: 14 }}
		>
			<Typography.Paragraph type="secondary">
				开启后，机器人会根据群聊内容主动判断是否参与对话，无需关键词或 @。群聊单独配置时会覆盖这里的全局设置。
			</Typography.Paragraph>
			<Form
				form={form}
				layout="vertical"
				initialValues={{
					free_reply_enabled: false,
					free_reply_level: 'normal',
					free_reply_cooldown_seconds: 60,
					free_reply_daily_limit: 30,
				}}
			>
				<Form.Item
					name="free_reply_enabled"
					label="自由回复"
					valuePropName="checked"
				>
					<Switch
						checkedChildren="开启"
						unCheckedChildren="关闭"
					/>
				</Form.Item>
				<Space
					align="start"
					wrap
					style={{ display: 'flex' }}
				>
					<Form.Item
						name="free_reply_level"
						label="参与频率"
						rules={[{ required: true }]}
						style={{ minWidth: 180, flex: 1 }}
					>
						<Select
							options={[
								{ label: '积极', value: 'active' },
								{ label: '平衡', value: 'normal' },
								{ label: '谨慎', value: 'cautious' },
							]}
						/>
					</Form.Item>
					<Form.Item
						name="free_reply_cooldown_seconds"
						label="同群冷却（秒）"
						rules={[{ required: true }]}
						style={{ minWidth: 180, flex: 1 }}
					>
						<InputNumber
							min={0}
							max={86400}
							precision={0}
							style={{ width: '100%' }}
						/>
					</Form.Item>
					<Form.Item
						name="free_reply_daily_limit"
						label="单群每日上限"
						rules={[{ required: true }]}
						style={{ minWidth: 180, flex: 1 }}
					>
						<InputNumber
							min={0}
							max={10000}
							precision={0}
							style={{ width: '100%' }}
						/>
					</Form.Item>
				</Space>
				<Button
					type="primary"
					icon={<SaveOutlined />}
					loading={loading}
					onClick={() => void save()}
				>
					保存自由回复
				</Button>
			</Form>
		</ParamsGroup>
	);
};

export default FreeReplySettings;
