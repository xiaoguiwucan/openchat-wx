import { useRequest } from 'ahooks';
import {
	Alert,
	App,
	Avatar,
	Button,
	Col,
	Drawer,
	Form,
	Input,
	InputNumber,
	Row,
	Select,
	Space,
	Spin,
	Switch,
} from 'antd';
import React from 'react';
import type * as Api from '@/api/wechat-robot/wechat-robot';
import { filterOption } from '@/common/filter-option';
import { maxTagPlaceholder } from '@/common/maxTagPlaceholder';
import ParamsGroup from '@/components/ParamsGroup';
import SystemPromptEditor from '@/components/SystemPromptEditor';
import { DefaultAvatar } from '@/constant';
import AIDrawingSettingsEditor from './AIDrawingSettingsEditor';
import AIPodcastConfigEditor from './AIPodcastConfigEditor';
import AIProviderModelFields from './AIProviderModelFields';
import { openchatAPIBase, openchatRequest } from './openchat-api';
import TTSettingsEditor from './TTSettingsEditor';
import { useAIProviderModels } from './useAIProviderModels';
import { imageRecognitionModelTips, ObjectToString, onTTSEnabledChange } from './utils';

interface IProps {
	robotId: number;
	robotCode: string;
	chatRoom: NonNullable<NonNullable<Api.Contact.ListList.ResponseBody['data']>['items']>[number];
	open: boolean;
	onClose: () => void;
}

type IFormValue = Api.ChatRoomSettings.ChatRoomSettingsCreate.RequestBody;

const ChatRoomSettings = (props: IProps) => {
	const { message } = App.useApp();

	const { chatRoom } = props;

	const [form] = Form.useForm<IFormValue>();
	const configIdRef = React.useRef(0);
	const { loading: providerLoading, providers } = useAIProviderModels({
		robotCode: props.robotCode,
		scope: 'chat_room',
		targetId: chatRoom.wechat_id!,
	});

	const { data: chatRoomMembers = [], loading: loadingChatRoomMembers } = useRequest(
		async () => {
			const resp = await window.wechatRobotClient.chatRoom.notLeftMembersList({
				id: props.robotId,
				chat_room_id: chatRoom.wechat_id!,
			});
			return resp.data?.data || [];
		},
		{
			manual: false,
			onError: reason => {
				message.error(reason.message);
			},
		},
	);

	// 加载全局配置
	const { data: globalSettings, loading: globalLoading } = useRequest(
		async () => {
			const resp = await window.wechatRobotClient.globalSettings.globalSettingsList({ id: props.robotId });
			return resp.data;
		},
		{
			manual: false,
			onError: reason => {
				message.error(reason.message);
			},
		},
	);

	// 知识库列表
	const { data: knowledgeCategories, loading: knowledgeCategoriesLoading } = useRequest(
		async () => {
			const resp = await window.wechatRobotClient.knowledge.categoriesList({
				id: props.robotId,
				type: 'text',
			});
			return (resp.data?.data || []).map(item => {
				return {
					label: item.name,
					value: item.code,
					text: `${item.name} (${item.code})`,
				};
			});
		},
		{
			manual: false,
			onError: reason => {
				message.error(reason.message);
			},
		},
	);

	// 加载群聊设置
	const chatRoomSettingsURL = `${openchatAPIBase(props.robotCode)}/chat-room-settings`;
	const { data, loading } = useRequest(
		async () => {
			return openchatRequest<Api.DtoGetChatRoomSettingsResponse>(
				`${chatRoomSettingsURL}?chat_room_id=${encodeURIComponent(chatRoom.wechat_id!)}`,
			);
		},
		{
			manual: false,
			onSuccess: resp => {
				if (!resp) {
					return;
				}
				const settings = { ...resp };
				configIdRef.current = settings.id || 0;
				settings.chat_ai_provider_id ??= 0;
				settings.image_recognition_provider_id ??= 0;
				settings.image_generation_provider_id ??= 0;
				settings.summary_ai_provider_id ??= 0;
				if (settings.image_ai_settings && typeof settings.image_ai_settings === 'object') {
					const legacy = settings.image_ai_settings as Record<string, unknown>;
					settings.image_generation_model ||= typeof legacy.model === 'string' ? legacy.model : undefined;
					settings.image_ai_settings = Object.fromEntries(
						['size', 'quality', 'response_format']
							.map(key => [key, legacy[key]])
							.filter(([, value]) => typeof value === 'string' && value),
					);
				}
				ObjectToString(settings);
				form.setFieldsValue(settings);
			},
			onError: reason => {
				message.error(reason.message);
			},
		},
	);

	const { runAsync: onSave, loading: saveLoading } = useRequest(
		async (data: Api.ChatRoomSettings.ChatRoomSettingsCreate.RequestBody) => {
			return openchatRequest<null>(chatRoomSettingsURL, {
				method: 'POST',
				body: JSON.stringify(data),
			});
		},
		{
			manual: true,
			onSuccess: () => {
				message.success('保存成功');
				props.onClose();
			},
			onError: reason => {
				message.error(reason.message);
			},
		},
	);

	const onOk = async () => {
		const values = await form.validateFields();
		for (const providerField of [
			'chat_ai_provider_id',
			'image_recognition_provider_id',
			'image_generation_provider_id',
			'summary_ai_provider_id',
		] as const) {
			values[providerField] = form.getFieldValue(providerField);
		}

		if (values.image_ai_enabled) {
			try {
				const raw = values.image_ai_settings as unknown;
				const json = typeof raw === 'string' && raw.trim() ? JSON.parse(raw) : raw || {};
				if (!json || typeof json !== 'object' || Array.isArray(json)) {
					message.error('绘图设置格式错误，不是有效的JSON对象格式');
					return;
				}
				values.image_ai_settings = Object.fromEntries(
					['size', 'quality', 'response_format']
						.map(key => [key, (json as Record<string, unknown>)[key]])
						.filter(([, value]) => typeof value === 'string' && value),
				);
			} catch {
				message.error('绘图设置格式错误，不是有效的JSON对象格式');
				return;
			}
		}
		if (values.tts_enabled) {
			try {
				const json = JSON.parse(values.tts_settings as unknown as string);
				if (!json || typeof json !== 'object' || Array.isArray(json)) {
					message.error('语音设置格式错误，不是有效的JSON对象格式');
					return;
				}
				values.tts_settings = json;
			} catch {
				message.error('语音设置格式错误，不是有效的JSON对象格式');
				return;
			}
		}
		if (values.podcast_config) {
			try {
				const json = JSON.parse(values.podcast_config as unknown as string);
				if (!json || typeof json !== 'object' || Array.isArray(json)) {
					message.error('播客设置格式错误，不是有效的JSON对象格式');
					return;
				}
				values.podcast_config = json;
			} catch {
				message.error('播客设置格式错误，不是有效的JSON对象格式');
				return;
			}
		} else {
			values.podcast_config = null as unknown as object;
		}
		if (values.wxhb_notify_member_list) {
			values.wxhb_notify_member_list = (values.wxhb_notify_member_list as unknown as string[]).join(',');
		}
		await onSave({ ...values, chat_room_id: chatRoom.wechat_id!, id: configIdRef.current });
	};

	const applyGlobalSettings = (
		type: 'chat' | 'free_reply' | 'drawing' | 'tts' | 'welcome' | 'pat' | 'leave_chat_room_alert' | 'all',
	) => {
		if (!globalSettings?.data) {
			message.error('全局配置不存在');
			return;
		}
		const imageAiSettings = globalSettings.data.image_ai_settings
			? JSON.stringify(globalSettings.data.image_ai_settings, null, 2)
			: '{}';
		const _ttsSettings = globalSettings.data.tts_settings
			? JSON.stringify(globalSettings.data.tts_settings, null, 2)
			: '{}';
		const chatSettings: Partial<IFormValue> = {
			chat_ai_enabled: globalSettings.data.chat_ai_enabled,
			chat_ai_trigger: globalSettings.data.chat_ai_trigger,
			chat_ai_provider_id: globalSettings.data.chat_ai_provider_id,
			chat_model: globalSettings.data.chat_model,
			image_recognition_provider_id: globalSettings.data.image_recognition_provider_id,
			image_recognition_model: globalSettings.data.image_recognition_model,
			max_completion_tokens: globalSettings.data.max_completion_tokens,
			chat_prompt: globalSettings.data.chat_prompt,
		};
		const freeReplySettings: Partial<IFormValue> = {
			free_reply_enabled: globalSettings.data.free_reply_enabled,
			free_reply_level: globalSettings.data.free_reply_level,
			free_reply_cooldown_seconds: globalSettings.data.free_reply_cooldown_seconds,
			free_reply_daily_limit: globalSettings.data.free_reply_daily_limit,
		};
		const globalImageSettings = (globalSettings.data.image_ai_settings || {}) as Record<string, unknown>;
		const drawingSettings: Partial<IFormValue> = {
			image_ai_enabled: globalSettings.data.image_ai_enabled,
			image_generation_provider_id: globalSettings.data.image_generation_provider_id,
			image_generation_model: typeof globalImageSettings.model === 'string' ? globalImageSettings.model : undefined,
			image_ai_settings: imageAiSettings as unknown as object,
		};
		const welcomeSettings: Partial<IFormValue> = {
			welcome_enabled: globalSettings.data.welcome_enabled,
			welcome_type: globalSettings.data.welcome_type,
			welcome_text: globalSettings.data.welcome_text,
			welcome_emoji_md5: globalSettings.data.welcome_emoji_md5,
			welcome_emoji_len: globalSettings.data.welcome_emoji_len,
			welcome_image_url: globalSettings.data.welcome_image_url,
			welcome_url: globalSettings.data.welcome_url,
		};
		const patSettings: Partial<IFormValue> = {
			pat_enabled: globalSettings.data.pat_enabled,
			pat_type: globalSettings.data.pat_type,
			pat_text: globalSettings.data.pat_text,
			pat_voice_timbre: globalSettings.data.pat_voice_timbre,
		};
		const ttsSettings: Partial<IFormValue> = {
			tts_enabled: globalSettings.data.tts_enabled,
			tts_model: globalSettings.data.tts_model,
			tts_settings: _ttsSettings as unknown as object,
		};
		const leaveChatRoomAlertSettings: Partial<IFormValue> = {
			leave_chat_room_alert_enabled: globalSettings.data.leave_chat_room_alert_enabled,
			leave_chat_room_alert_text: globalSettings.data.leave_chat_room_alert_text,
		};
		const otherSettings: Partial<IFormValue> = {
			chat_room_ranking_enabled: globalSettings.data.chat_room_ranking_enabled,
			chat_room_summary_enabled: globalSettings.data.chat_room_summary_enabled,
			summary_ai_provider_id: globalSettings.data.summary_ai_provider_id,
			chat_room_summary_model: globalSettings.data.chat_room_summary_model,
			chat_room_summary_mode: globalSettings.data.chat_room_summary_mode,
			news_enabled: globalSettings.data.news_enabled,
			news_type: globalSettings.data.news_type,
			morning_enabled: globalSettings.data.morning_enabled,
		};
		switch (type) {
			case 'chat':
				form.setFieldsValue(chatSettings);
				break;
			case 'free_reply':
				form.setFieldsValue(freeReplySettings);
				break;
			case 'drawing':
				form.setFieldsValue(drawingSettings);
				break;
			case 'tts':
				form.setFieldsValue(ttsSettings);
				break;
			case 'welcome':
				form.setFieldsValue(welcomeSettings);
				break;
			case 'pat':
				form.setFieldsValue(patSettings);
				break;
			case 'leave_chat_room_alert':
				form.setFieldsValue(leaveChatRoomAlertSettings);
				break;
			case 'all':
				form.setFieldsValue({
					...chatSettings,
					...freeReplySettings,
					...drawingSettings,
					...ttsSettings,
					...welcomeSettings,
					...patSettings,
					...leaveChatRoomAlertSettings,
					...otherSettings,
				});
				break;
			default:
				message.error('未知类型');
				return;
		}
	};

	const getChatRoomMemberSelector = (placeholder: string) => {
		return (
			<Select
				style={{ width: '100%' }}
				mode="multiple"
				placeholder={placeholder}
				showSearch={{
					filterOption,
				}}
				allowClear
				loading={loadingChatRoomMembers}
				options={chatRoomMembers.map(item => {
					const labelText = item.remark || item.nickname || item.alias || item.wechat_id;
					return {
						label: (
							<Row
								align="middle"
								wrap={false}
								gutter={3}
							>
								<Col flex="0 0 auto">
									<Avatar
										src={item.avatar || DefaultAvatar}
										gap={0}
										size={18}
									/>
								</Col>
								<Col
									flex="1 1 auto"
									className="ellipsis"
								>
									{labelText}
								</Col>
							</Row>
						),
						value: item.wechat_id,
						text: `${item.remark || ''} ${item.nickname || ''} ${item.alias || ''} ${item.wechat_id}`,
					};
				})}
			/>
		);
	};

	return (
		<Drawer
			title={
				<Row
					align="middle"
					wrap={false}
				>
					<Col flex="0 0 32px">
						<Avatar src={chatRoom.avatar || DefaultAvatar} />
					</Col>
					<Col
						flex="0 1 auto"
						className="ellipsis"
						style={{ padding: '0 3px' }}
					>
						{chatRoom.remark || chatRoom.alias || chatRoom.nickname || chatRoom.wechat_id} 聊天设置
						{data?.id === 0 && (
							<span style={{ fontSize: 12, color: '#ff5722' }}>(当前群聊未进行过任何设置，运行时会继承全局设置)</span>
						)}
					</Col>
				</Row>
			}
			extra={
				<Space>
					<Button
						type="primary"
						loading={globalLoading}
						onClick={() => applyGlobalSettings('all')}
					>
						使用全局配置填充
					</Button>
				</Space>
			}
			open={props.open}
			onClose={props.onClose}
			size="min(99vw, 900px)"
			styles={{
				header: { paddingTop: 12, paddingBottom: 12 },
				body: { paddingTop: 16, paddingBottom: 0 },
				footer: { padding: 0 },
			}}
			footer={
				<Row style={{ overflow: 'hidden' }}>
					<Col
						span={12}
						style={{ borderRight: '1px solid #f0f0f0' }}
					>
						<Button
							size="large"
							type="text"
							block
							onClick={props.onClose}
						>
							取消
						</Button>
					</Col>
					<Col span={12}>
						<Button
							size="large"
							type="primary"
							block
							style={{ borderRadius: 0 }}
							loading={saveLoading}
							onClick={onOk}
						>
							确认
						</Button>
					</Col>
				</Row>
			}
		>
			<Spin spinning={loading || globalLoading || providerLoading}>
				<Form
					layout="vertical"
					form={form}
					labelWrap
					autoComplete="off"
					scrollToFirstError={{ behavior: 'instant', block: 'end', focus: true }}
				>
					<Form.Item
						name="id"
						hidden
					>
						<Input />
					</Form.Item>
					<ParamsGroup
						title="AI聊天设置"
						style={{ marginTop: 10 }}
					>
						<Form.Item
							layout="horizontal"
							name="chat_ai_enabled"
							label="聊天AI"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
							/>
						</Form.Item>
						<Form.Item
							noStyle
							shouldUpdate={(prev: IFormValue, next: IFormValue) => prev.chat_ai_enabled !== next.chat_ai_enabled}
						>
							{({ getFieldValue }) => {
								if (getFieldValue('chat_ai_enabled')) {
									return (
										<>
											<Form.Item
												name="chat_ai_trigger"
												label="强制唤醒词（可选）"
												tooltip="消息以该词开头时一定触发 AI；自由回复开启后，无需该词也可由机器人主动参与"
											>
												<Input
													placeholder="留空则继承全局强制唤醒词"
													allowClear
												/>
											</Form.Item>
											<Alert
												showIcon
												type="info"
												title="群聊可覆盖全局模型"
												description="连接地址和密钥由所选模型渠道自动提供；渠道留空时继承全局设置。"
												style={{ marginBottom: 16 }}
											/>
											<AIProviderModelFields
												form={form}
												providers={providers}
												providerName="chat_ai_provider_id"
												modelName="chat_model"
												providerLabel="AI回复渠道"
												modelLabel="聊天模型"
												defaultModelKey="chat_model"
												required={false}
												allowInherit
											/>
											<AIProviderModelFields
												form={form}
												providers={providers}
												providerName="image_recognition_provider_id"
												modelName="image_recognition_model"
												providerLabel="图像识别渠道"
												modelLabel="图像识别模型"
												defaultModelKey="image_recognition_model"
												modelTooltip={imageRecognitionModelTips}
												required={false}
												allowInherit
											/>
											<Form.Item
												name="knowledge_categories"
												label="绑定知识库"
												tooltip="绑定知识库后，AI 会按需从知识库中检索相关内容，并在回答中引用这些内容，提升回答的准确性和专业性"
											>
												<Select
													mode="multiple"
													placeholder="请选择知识库"
													showSearch={{
														filterOption,
													}}
													allowClear
													loading={knowledgeCategoriesLoading}
													maxTagCount="responsive"
													maxTagPlaceholder={maxTagPlaceholder}
													options={knowledgeCategories || []}
												/>
											</Form.Item>
											<Form.Item
												name="memory_extraction_blacklist"
												label="记忆提取黑名单"
												tooltip="屏蔽指定群成员的消息，他的消息不会提取到记忆中，比如群里的水军或者广告号，减少记忆噪音"
											>
												{getChatRoomMemberSelector('请选择记忆提取黑名单成员')}
											</Form.Item>
											<Form.Item
												name="max_completion_tokens"
												label="最大回复"
												tooltip="AI每次回复的最大词元个数，为0则表示不限制"
											>
												<InputNumber
													placeholder="请输入最大回复，为0则表示不限制"
													style={{ width: '100%' }}
													max={4096}
													min={0}
												/>
											</Form.Item>
											<Form.Item
												name="chat_prompt"
												label="人设"
												tooltip="人设是指在与AI进行对话时，系统会自动添加的提示信息，用于引导AI的回答方向和风格。"
											>
												<SystemPromptEditor robotId={props.robotId} />
											</Form.Item>
										</>
									);
								}
								return null;
							}}
						</Form.Item>
						<Form.Item style={{ marginBottom: 6 }}>
							<div style={{ display: 'flex', justifyContent: 'flex-end' }}>
								<Button
									disabled={globalLoading}
									onClick={() => applyGlobalSettings('chat')}
								>
									使用全局配置填充AI聊天设置
								</Button>
							</div>
						</Form.Item>
					</ParamsGroup>
					<ParamsGroup
						title="自由回复设置"
						style={{ marginTop: 24 }}
					>
						<Form.Item
							layout="horizontal"
							name="free_reply_enabled"
							label="自由参与群聊"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
								onChange={(checked: boolean) => {
									if (checked && !form.getFieldValue('free_reply_level')) {
										form.setFieldsValue({
											free_reply_level: globalSettings?.data?.free_reply_level || 'normal',
											free_reply_cooldown_seconds: globalSettings?.data?.free_reply_cooldown_seconds ?? 60,
											free_reply_daily_limit: globalSettings?.data?.free_reply_daily_limit ?? 30,
										});
									}
								}}
							/>
						</Form.Item>
						<Form.Item
							noStyle
							shouldUpdate={(prev: IFormValue, next: IFormValue) => prev.free_reply_enabled !== next.free_reply_enabled}
						>
							{({ getFieldValue }) =>
								getFieldValue('free_reply_enabled') ? (
									<>
										<Form.Item
											name="free_reply_level"
											label="参与频率"
											rules={[{ required: true, message: '请选择自由回复参与频率' }]}
											help="按 LightAgent 评分档位控制参与频率；高频最容易接话，安静只回应高分内容。"
										>
											<Select
												options={[
													{ label: '高频', value: 'crazy' },
													{ label: '活跃', value: 'active' },
													{ label: '普通', value: 'normal' },
													{ label: '安静', value: 'cautious' },
												]}
											/>
										</Form.Item>
										<Row gutter={12}>
											<Col
												xs={24}
												sm={12}
											>
												<Form.Item
													name="free_reply_cooldown_seconds"
													label="回复冷却（秒）"
													tooltip="同一群两次自由回复之间的最短间隔，0 表示不限制"
													rules={[{ required: true, message: '请输入回复冷却时间' }]}
												>
													<InputNumber
														min={0}
														max={86400}
														precision={0}
														style={{ width: '100%' }}
													/>
												</Form.Item>
											</Col>
											<Col
												xs={24}
												sm={12}
											>
												<Form.Item
													name="free_reply_daily_limit"
													label="每日上限（次）"
													tooltip="这个群每天最多触发的自由回复次数，0 表示不限制"
													rules={[{ required: true, message: '请输入每日回复上限' }]}
												>
													<InputNumber
														min={0}
														max={10000}
														precision={0}
														style={{ width: '100%' }}
													/>
												</Form.Item>
											</Col>
										</Row>
									</>
								) : null
							}
						</Form.Item>
						<Form.Item style={{ marginBottom: 6 }}>
							<div style={{ display: 'flex', justifyContent: 'flex-end' }}>
								<Button
									disabled={globalLoading}
									onClick={() => applyGlobalSettings('free_reply')}
								>
									使用全局自由回复设置
								</Button>
							</div>
						</Form.Item>
					</ParamsGroup>
					<ParamsGroup
						title="AI绘图设置"
						style={{ marginTop: 24 }}
					>
						<Form.Item
							layout="horizontal"
							name="image_ai_enabled"
							label="AI 绘图"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
							/>
						</Form.Item>
						<Form.Item
							noStyle
							shouldUpdate={(prev: IFormValue, next: IFormValue) => prev.image_ai_enabled !== next.image_ai_enabled}
						>
							{({ getFieldValue }) => {
								if (getFieldValue('image_ai_enabled')) {
									return (
										<>
											<AIProviderModelFields
												form={form}
												providers={providers}
												providerName="image_generation_provider_id"
												modelName="image_generation_model"
												providerLabel="AI绘图渠道"
												modelLabel="生图模型"
												defaultModelKey="image_generation_model"
												required={false}
												allowInherit
											/>
											<Form.Item
												name="image_ai_settings"
												label="绘图参数"
											>
												<AIDrawingSettingsEditor channelManaged />
											</Form.Item>
										</>
									);
								}
								return null;
							}}
						</Form.Item>
						<Form.Item style={{ marginBottom: 6 }}>
							<div style={{ display: 'flex', justifyContent: 'flex-end' }}>
								<Button
									disabled={globalLoading}
									onClick={() => applyGlobalSettings('drawing')}
								>
									使用全局配置填充AI绘图设置
								</Button>
							</div>
						</Form.Item>
					</ParamsGroup>
					<ParamsGroup
						title="AI文本转语音设置"
						style={{ marginTop: 24 }}
					>
						<Form.Item
							layout="horizontal"
							name="tts_enabled"
							label="文本转语音"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
								onChange={(checked: boolean) => {
									onTTSEnabledChange(form, checked);
								}}
							/>
						</Form.Item>
						<Form.Item
							noStyle
							shouldUpdate={(prev: IFormValue, next: IFormValue) => prev.tts_enabled !== next.tts_enabled}
						>
							{({ getFieldValue }) => {
								if (getFieldValue('tts_enabled')) {
									return (
										<>
											<Form.Item
												name="tts_model"
												label="语音模型"
												rules={[{ required: true, message: '语音模型不能为空' }]}
											>
												<Select
													placeholder="请选择语音模型"
													style={{ width: '100%' }}
													options={[
														{ label: '豆包', value: 'doubao' },
														{ label: '小米', value: 'mimo' },
													]}
												/>
											</Form.Item>
											<Form.Item
												name="tts_settings"
												label="语音设置"
												rules={[{ required: true, message: '语音设置不能为空' }]}
												tooltip={
													<>
														<a
															target="_blank"
															rel="noreferrer"
															href="https://www.volcengine.com/docs/6561/1598757?lang=zh"
														>
															语音设置文档
														</a>
													</>
												}
											>
												<TTSettingsEditor />
											</Form.Item>
										</>
									);
								}
								return null;
							}}
						</Form.Item>
						<Form.Item style={{ marginBottom: 6 }}>
							<div style={{ display: 'flex', justifyContent: 'flex-end' }}>
								<Button
									disabled={globalLoading}
									onClick={() => applyGlobalSettings('tts')}
								>
									使用全局配置填充AI文本转语音设置
								</Button>
							</div>
						</Form.Item>
					</ParamsGroup>
					<ParamsGroup
						title="群聊欢迎新成员设置"
						style={{ marginTop: 24 }}
					>
						<Form.Item
							layout="horizontal"
							name="welcome_enabled"
							label="欢迎新成员"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
							/>
						</Form.Item>
						<Form.Item
							noStyle
							shouldUpdate={(prev: IFormValue, next: IFormValue) => prev.welcome_enabled !== next.welcome_enabled}
						>
							{({ getFieldValue }) => {
								if (getFieldValue('welcome_enabled')) {
									return (
										<>
											<Form.Item
												name="welcome_type"
												label="欢迎形式"
												rules={[{ required: true, message: '欢迎形式不能为空' }]}
											>
												<Select
													placeholder="请选择欢迎形式"
													style={{ width: '100%' }}
													options={[
														{ label: '纯文字', value: 'text' },
														{ label: '表情包', value: 'emoji' },
														{ label: '图片', value: 'image' },
														{ label: '卡片', value: 'url' },
													]}
												/>
											</Form.Item>
											<Form.Item
												noStyle
												shouldUpdate={(prev: IFormValue, next: IFormValue) => prev.welcome_type !== next.welcome_type}
											>
												{({ getFieldValue }) => {
													const type = getFieldValue('welcome_type');
													if (type === 'text') {
														return (
															<>
																<Form.Item
																	name="welcome_text"
																	label="欢迎语"
																	rules={[{ required: true, message: '欢迎语不能为空' }]}
																>
																	<Input
																		placeholder="请输入欢迎语"
																		allowClear
																	/>
																</Form.Item>
															</>
														);
													}
													if (type === 'emoji') {
														return (
															<>
																<Form.Item
																	name="welcome_emoji_md5"
																	label="表情包MD5"
																	rules={[{ required: true, message: '表情包MD5不能为空' }]}
																>
																	<Input
																		placeholder="请输入表情包MD5"
																		allowClear
																	/>
																</Form.Item>
																<Form.Item
																	name="welcome_emoji_len"
																	label="表情包长度"
																	rules={[{ required: true, message: '表情包长度不能为空' }]}
																>
																	<InputNumber
																		placeholder="请输入表情包长度"
																		min={1}
																		precision={0}
																		style={{ width: '100%' }}
																	/>
																</Form.Item>
															</>
														);
													}
													if (type === 'image') {
														return (
															<Form.Item
																name="welcome_image_url"
																label="图片地址"
																rules={[{ required: true, message: '图片地址不能为空' }]}
															>
																<Input
																	placeholder="请输入图片地址"
																	allowClear
																/>
															</Form.Item>
														);
													}
													if (type === 'url') {
														return (
															<>
																<Form.Item
																	name="welcome_text"
																	label="欢迎语"
																	rules={[{ required: true, message: '欢迎语不能为空' }]}
																>
																	<Input
																		placeholder="请输入欢迎语"
																		allowClear
																	/>
																</Form.Item>
																<Form.Item
																	name="welcome_url"
																	label="链接地址"
																	rules={[{ required: true, message: '链接地址不能为空' }]}
																>
																	<Input
																		placeholder="请输入链接地址"
																		allowClear
																	/>
																</Form.Item>
															</>
														);
													}
													return null;
												}}
											</Form.Item>
										</>
									);
								}
								return null;
							}}
						</Form.Item>
						<Form.Item style={{ marginBottom: 6 }}>
							<div style={{ display: 'flex', justifyContent: 'flex-end' }}>
								<Button
									disabled={globalLoading}
									onClick={() => applyGlobalSettings('welcome')}
								>
									使用全局配置填充群聊欢迎新成员设置
								</Button>
							</div>
						</Form.Item>
					</ParamsGroup>
					<ParamsGroup
						title="群聊短视频解析设置"
						style={{ marginTop: 24 }}
					>
						<Form.Item
							layout="horizontal"
							name="short_video_parsing_enabled"
							label="短视频解析"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
							/>
						</Form.Item>
					</ParamsGroup>
					<ParamsGroup
						title="群聊 AI 播客设置"
						style={{ marginTop: 24 }}
					>
						<Form.Item
							layout="horizontal"
							name="podcast_enabled"
							label="AI 播客"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
							/>
						</Form.Item>
						<Form.Item
							noStyle
							shouldUpdate={(prev: IFormValue, next: IFormValue) => prev.podcast_enabled !== next.podcast_enabled}
						>
							{({ getFieldValue }) => {
								const enabled = getFieldValue('podcast_enabled');
								return (
									<Form.Item
										name="podcast_config"
										label="播客设置"
										rules={[{ required: enabled, message: '播客设置不能为空' }]}
									>
										<AIPodcastConfigEditor />
									</Form.Item>
								);
							}}
						</Form.Item>
					</ParamsGroup>
					<ParamsGroup
						title="群聊红包提醒设置"
						style={{ marginTop: 24 }}
					>
						<Form.Item
							layout="horizontal"
							name="wxhb_notify_enabled"
							label="红包提醒"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
							/>
						</Form.Item>
						<Form.Item
							noStyle
							shouldUpdate={(prev: IFormValue, next: IFormValue) =>
								prev.wxhb_notify_enabled !== next.wxhb_notify_enabled
							}
						>
							{({ getFieldValue }) => {
								const enabled = getFieldValue('wxhb_notify_enabled');
								return (
									<Form.Item
										name="wxhb_notify_member_list"
										label="提醒人"
										rules={[{ required: enabled, message: '提醒人不能为空' }]}
									>
										{getChatRoomMemberSelector('请选择提醒人')}
									</Form.Item>
								);
							}}
						</Form.Item>
					</ParamsGroup>
					<ParamsGroup
						title="群聊拍一拍设置"
						style={{ marginTop: 24 }}
					>
						<Form.Item
							layout="horizontal"
							name="pat_enabled"
							label="拍一拍"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
							/>
						</Form.Item>
						<Form.Item
							noStyle
							shouldUpdate={(prev: IFormValue, next: IFormValue) => prev.pat_enabled !== next.pat_enabled}
						>
							{({ getFieldValue }) => {
								if (getFieldValue('pat_enabled')) {
									return (
										<>
											<Form.Item
												name="pat_type"
												label="交互类型"
												rules={[{ required: true, message: '交互类型不能为空' }]}
											>
												<Select
													placeholder="请选择交互类型"
													style={{ width: '100%' }}
													options={[
														{ label: '文字', value: 'text' },
														{ label: '语音', value: 'voice' },
													]}
												/>
											</Form.Item>
											<Form.Item
												noStyle
												shouldUpdate={(prev: IFormValue, next: IFormValue) => prev.pat_type !== next.pat_type}
											>
												{({ getFieldValue }) => {
													if (getFieldValue('pat_type') === 'voice') {
														return (
															<Form.Item
																name="pat_voice_timbre"
																label="语音音色"
																rules={[{ required: true, message: '语音音色不能为空' }]}
															>
																<Input
																	placeholder="请输入语音音色"
																	allowClear
																/>
															</Form.Item>
														);
													}
													return null;
												}}
											</Form.Item>
											<Form.Item
												name="pat_text"
												label="文字"
												rules={[
													{ required: true, message: '文本语言不能为空' },
													{ max: 255, message: '文字不能超过255个字符' },
												]}
											>
												<Input
													placeholder="请输入文字，为语音的时候，则是文字转语音"
													allowClear
												/>
											</Form.Item>
										</>
									);
								}
								return null;
							}}
						</Form.Item>
						<Form.Item style={{ marginBottom: 6 }}>
							<div style={{ display: 'flex', justifyContent: 'flex-end' }}>
								<Button
									disabled={globalLoading}
									onClick={() => applyGlobalSettings('pat')}
								>
									使用全局配置填充群聊拍一拍设置
								</Button>
							</div>
						</Form.Item>
					</ParamsGroup>
					<ParamsGroup
						title="群聊退群提醒设置"
						style={{ marginTop: 24 }}
					>
						<Form.Item
							layout="horizontal"
							name="leave_chat_room_alert_enabled"
							label="退群提醒"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
								onChange={(checked: boolean) => {
									if (checked && !form.getFieldValue('leave_chat_room_alert_text')) {
										form.setFieldsValue({
											leave_chat_room_alert_text: '阿拉蕾，{placeholder}退出了群聊',
										});
									}
								}}
							/>
						</Form.Item>
						<Form.Item
							noStyle
							shouldUpdate={(prev: IFormValue, next: IFormValue) =>
								prev.leave_chat_room_alert_enabled !== next.leave_chat_room_alert_enabled
							}
						>
							{({ getFieldValue }) => {
								if (getFieldValue('leave_chat_room_alert_enabled')) {
									return (
										<>
											<Form.Item
												name="leave_chat_room_alert_text"
												label="提醒文本"
												rules={[{ required: true, message: '提醒文本不能为空' }]}
											>
												<Input
													placeholder="请输入提醒文本"
													allowClear
												/>
											</Form.Item>
										</>
									);
								}
								return null;
							}}
						</Form.Item>
						<Form.Item style={{ marginBottom: 6 }}>
							<div style={{ display: 'flex', justifyContent: 'flex-end' }}>
								<Button
									disabled={globalLoading}
									onClick={() => applyGlobalSettings('leave_chat_room_alert')}
								>
									使用全局配置填充群聊退群提醒设置
								</Button>
							</div>
						</Form.Item>
					</ParamsGroup>
					<ParamsGroup
						title="群聊排行榜设置"
						style={{ marginTop: 24 }}
					>
						<>
							{!globalSettings?.data?.chat_room_ranking_enabled && (
								<Alert
									style={{ marginTop: 10, marginBottom: 10 }}
									type="warning"
									title={<>全局设置下面的群聊排行榜设置未开启，当前设置将不会生效</>}
								/>
							)}
						</>
						<Form.Item
							layout="horizontal"
							name="chat_room_ranking_enabled"
							label="排行榜"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
							/>
						</Form.Item>
					</ParamsGroup>
					<ParamsGroup
						title="群聊总结设置"
						style={{ marginTop: 24 }}
					>
						<>
							{!globalSettings?.data?.chat_room_summary_enabled && (
								<Alert
									style={{ marginTop: 10, marginBottom: 10 }}
									type="warning"
									title={<>全局设置下面的群聊总结设置未开启，当前设置将不会生效</>}
								/>
							)}
						</>
						<Form.Item
							layout="horizontal"
							name="chat_room_summary_enabled"
							label="群聊总结"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
							/>
						</Form.Item>
						<Form.Item
							noStyle
							shouldUpdate={(prev: IFormValue, next: IFormValue) =>
								prev.chat_room_summary_enabled !== next.chat_room_summary_enabled
							}
						>
							{({ getFieldValue }) => {
								if (getFieldValue('chat_room_summary_enabled')) {
									return (
										<>
											<AIProviderModelFields
												form={form}
												providers={providers}
												providerName="summary_ai_provider_id"
												modelName="chat_room_summary_model"
												providerLabel="群聊总结渠道"
												modelLabel="群聊总结模型"
												defaultModelKey="summary_model"
												required={false}
												allowInherit
											/>
											<Form.Item
												name="chat_room_summary_mode"
												label="显示模式"
											>
												<Select
													placeholder="请选择显示模式"
													style={{ width: '100%' }}
													options={[
														{ label: '文本', value: 'text' },
														{ label: '图片', value: 'image' },
													]}
												/>
											</Form.Item>
										</>
									);
								}
								return null;
							}}
						</Form.Item>
					</ParamsGroup>
					<ParamsGroup
						title="每日早报设置"
						style={{ marginTop: 24 }}
					>
						<>
							{!globalSettings?.data?.news_enabled && (
								<Alert
									style={{ marginTop: 10, marginBottom: 10 }}
									type="warning"
									title={<>全局设置下面的每日早报设置未开启，当前设置将不会生效</>}
								/>
							)}
						</>
						<Form.Item
							layout="horizontal"
							name="news_enabled"
							label="每日早报"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
							/>
						</Form.Item>
						<Form.Item
							noStyle
							shouldUpdate={(prev: IFormValue, next: IFormValue) => prev.news_enabled !== next.news_enabled}
						>
							{({ getFieldValue }) => {
								if (getFieldValue('news_enabled')) {
									return (
										<>
											<Form.Item
												name="news_type"
												label="早报类型"
											>
												<Select
													placeholder="请选择早报类型"
													style={{ width: '100%' }}
													allowClear
													options={[
														{ label: '文字', value: 'text' },
														{ label: '图片', value: 'image' },
													]}
												/>
											</Form.Item>
										</>
									);
								}
								return null;
							}}
						</Form.Item>
					</ParamsGroup>
					<ParamsGroup
						title="每日早安设置"
						style={{ marginTop: 24 }}
					>
						<>
							{!globalSettings?.data?.morning_enabled && (
								<Alert
									style={{ marginTop: 10, marginBottom: 10 }}
									type="warning"
									title={<>全局设置下面的每日早安设置未开启，当前设置将不会生效</>}
								/>
							)}
						</>
						<Form.Item
							layout="horizontal"
							name="morning_enabled"
							label="每日早安"
							valuePropName="checked"
						>
							<Switch
								unCheckedChildren="关闭"
								checkedChildren="开启"
							/>
						</Form.Item>
					</ParamsGroup>
				</Form>
			</Spin>
		</Drawer>
	);
};

export default React.memo(ChatRoomSettings);
