package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/xiaoguiwucan/openchat-wx/interface/settings"
	"github.com/xiaoguiwucan/openchat-wx/model"
	"github.com/xiaoguiwucan/openchat-wx/pkg/robot"
	"github.com/xiaoguiwucan/openchat-wx/repository"
	"github.com/xiaoguiwucan/openchat-wx/utils"
	"github.com/xiaoguiwucan/openchat-wx/vars"
	"gorm.io/datatypes"
)

type ChatRoomSettingsService struct {
	ctx              context.Context
	Message          *model.Message
	gsRepo           *repository.GlobalSettings
	crsRepo          *repository.ChatRoomSettings
	globalSettings   *model.GlobalSettings
	chatRoomSettings *model.ChatRoomSettings
	chatProvider     *model.AIProvider
	visionProvider   *model.AIProvider
	imageProvider    *model.AIProvider
	isFreeReply      bool
}

var _ settings.Settings = (*ChatRoomSettingsService)(nil)

func NewChatRoomSettingsService(ctx context.Context) *ChatRoomSettingsService {
	return &ChatRoomSettingsService{
		ctx:     ctx,
		gsRepo:  repository.NewGlobalSettingsRepo(ctx, vars.DB),
		crsRepo: repository.NewChatRoomSettingsRepo(ctx, vars.DB),
	}
}

func (s *ChatRoomSettingsService) GetChatRoomSettings(chatRoomID string) (*model.ChatRoomSettings, error) {
	return s.crsRepo.GetChatRoomSettings(chatRoomID)
}

func (s *ChatRoomSettingsService) InitByMessage(message *model.Message) error {
	s.Message = message
	globalSettings, err := s.gsRepo.GetGlobalSettings()
	if err != nil {
		return err
	}
	s.globalSettings = globalSettings
	chatRoomSettings, err := s.crsRepo.GetChatRoomSettings(message.FromWxID)
	if err != nil {
		return err
	}
	s.chatRoomSettings = chatRoomSettings
	var legacyOverrideID, chatOverrideID, visionOverrideID, imageOverrideID *int64
	if chatRoomSettings != nil {
		legacyOverrideID = chatRoomSettings.AIProviderID
		chatOverrideID = chatRoomSettings.ChatAIProviderID
		visionOverrideID = chatRoomSettings.ImageRecognitionProviderID
		imageOverrideID = chatRoomSettings.ImageGenerationProviderID
	}
	s.chatProvider, s.visionProvider, s.imageProvider, err = ResolveAIProvidersForTarget(
		s.ctx, globalSettings, legacyOverrideID, chatOverrideID, visionOverrideID, imageOverrideID,
	)
	if err != nil {
		return fmt.Errorf("加载模型渠道失败: %w", err)
	}
	return nil
}

func (s *ChatRoomSettingsService) GetAIConfig() settings.AIConfig {
	aiConfig := settings.AIConfig{}
	if s.globalSettings != nil {
		if s.globalSettings.ChatBaseURL != "" {
			aiConfig.BaseURL = s.globalSettings.ChatBaseURL
		}
		if s.globalSettings.ChatAPIKey != "" {
			aiConfig.APIKey = s.globalSettings.ChatAPIKey
		}
		if s.globalSettings.ChatModel != "" {
			aiConfig.Model = s.globalSettings.ChatModel
		}
		if s.globalSettings.ImageRecognitionModel != "" {
			aiConfig.ImageRecognitionModel = s.globalSettings.ImageRecognitionModel
		}
		if s.globalSettings.ChatPrompt != "" {
			aiConfig.Prompt = s.globalSettings.ChatPrompt
		}
		if s.globalSettings.MaxCompletionTokens != nil {
			aiConfig.MaxCompletionTokens = *s.globalSettings.MaxCompletionTokens
		}
		if s.globalSettings.ImageAISettings != nil {
			aiConfig.ImageAISettings = s.globalSettings.ImageAISettings
		}
		if s.globalSettings.TTSModel != nil && *s.globalSettings.TTSModel != "" {
			aiConfig.TTSModel = *s.globalSettings.TTSModel
		}
		if s.globalSettings.TTSSettings != nil {
			aiConfig.TTSSettings = s.globalSettings.TTSSettings
		}
	}
	if s.chatRoomSettings != nil {
		if s.chatRoomSettings.ChatBaseURL != nil && *s.chatRoomSettings.ChatBaseURL != "" {
			aiConfig.BaseURL = *s.chatRoomSettings.ChatBaseURL
		}
		if s.chatRoomSettings.ChatAPIKey != nil && *s.chatRoomSettings.ChatAPIKey != "" {
			aiConfig.APIKey = *s.chatRoomSettings.ChatAPIKey
		}
		if s.chatRoomSettings.ChatModel != nil && *s.chatRoomSettings.ChatModel != "" {
			aiConfig.Model = *s.chatRoomSettings.ChatModel
		}
		if s.chatRoomSettings.ImageRecognitionModel != nil && *s.chatRoomSettings.ImageRecognitionModel != "" {
			aiConfig.ImageRecognitionModel = *s.chatRoomSettings.ImageRecognitionModel
		}
		if s.chatRoomSettings.ChatPrompt != nil && *s.chatRoomSettings.ChatPrompt != "" {
			aiConfig.Prompt = *s.chatRoomSettings.ChatPrompt
		}
		if s.chatRoomSettings.MaxCompletionTokens != nil {
			aiConfig.MaxCompletionTokens = *s.chatRoomSettings.MaxCompletionTokens
		}
		if s.chatRoomSettings.ImageAISettings != nil {
			mergedImageSettings := map[string]any{}
			_ = json.Unmarshal(aiConfig.ImageAISettings, &mergedImageSettings)
			roomImageSettings := map[string]any{}
			_ = json.Unmarshal(s.chatRoomSettings.ImageAISettings, &roomImageSettings)
			for key, value := range roomImageSettings {
				mergedImageSettings[key] = value
			}
			if encoded, err := json.Marshal(mergedImageSettings); err == nil {
				aiConfig.ImageAISettings = datatypes.JSON(encoded)
			}
		}
		if s.chatRoomSettings.ImageGenerationModel != nil && strings.TrimSpace(*s.chatRoomSettings.ImageGenerationModel) != "" {
			imageSettings := map[string]any{}
			_ = json.Unmarshal(aiConfig.ImageAISettings, &imageSettings)
			imageSettings["model"] = strings.TrimSpace(*s.chatRoomSettings.ImageGenerationModel)
			if encoded, err := json.Marshal(imageSettings); err == nil {
				aiConfig.ImageAISettings = datatypes.JSON(encoded)
			}
		}
		if s.chatRoomSettings.TTSModel != nil && *s.chatRoomSettings.TTSModel != "" {
			aiConfig.TTSModel = *s.chatRoomSettings.TTSModel
		}
		if s.chatRoomSettings.TTSSettings != nil {
			aiConfig.TTSSettings = s.chatRoomSettings.TTSSettings
		}
	}
	ApplyAIProviders(&aiConfig, s.chatProvider, s.visionProvider, s.imageProvider)
	aiConfig.BaseURL = utils.NormalizeAIBaseURL(aiConfig.BaseURL)
	if s.Message != nil {
		chatProviderName := "legacy"
		if s.chatProvider != nil {
			chatProviderName = s.chatProvider.Name
		}
		log.Printf("[AIConfig] from=%s chat_provider=%s chat_model=%s", s.Message.FromWxID, chatProviderName, aiConfig.Model)
	}
	return aiConfig
}

func (s *ChatRoomSettingsService) IsAIChatEnabled() bool {
	if s.chatRoomSettings != nil && s.chatRoomSettings.ChatAIEnabled != nil {
		return *s.chatRoomSettings.ChatAIEnabled
	}
	if s.globalSettings != nil && s.globalSettings.ChatAIEnabled != nil {
		return *s.globalSettings.ChatAIEnabled
	}
	return false
}

func (s *ChatRoomSettingsService) IsAIDrawingEnabled() bool {
	if s.chatRoomSettings != nil && s.chatRoomSettings.ImageAIEnabled != nil {
		return *s.chatRoomSettings.ImageAIEnabled
	}
	if s.globalSettings != nil && s.globalSettings.ImageAIEnabled != nil {
		return *s.globalSettings.ImageAIEnabled
	}
	return false
}

func (s *ChatRoomSettingsService) IsTTSEnabled() bool {
	if s.chatRoomSettings != nil && s.chatRoomSettings.TTSEnabled != nil {
		return *s.chatRoomSettings.TTSEnabled
	}
	if s.globalSettings != nil && s.globalSettings.TTSEnabled != nil {
		return *s.globalSettings.TTSEnabled
	}
	return false
}

func (s *ChatRoomSettingsService) IsShortVideoParsingEnabled() bool {
	if s.chatRoomSettings != nil && s.chatRoomSettings.ShortVideoParsingEnabled != nil {
		return *s.chatRoomSettings.ShortVideoParsingEnabled
	}
	return false
}

func (s *ChatRoomSettingsService) logAITrigger(reason, triggerWord, messageContent string) {
	if s.Message == nil {
		return
	}
	contentPreview := strings.ReplaceAll(strings.TrimSpace(messageContent), "\n", `\n`)
	contentRunes := []rune(contentPreview)
	if len(contentRunes) > 80 {
		contentPreview = string(contentRunes[:80]) + "..."
	}
	log.Printf("[AITrigger] reason=%s trigger_word=%q msg_id=%d from=%s sender=%s is_at_me=%t app_msg_type=%d content=%q",
		reason,
		triggerWord,
		s.Message.MsgId,
		s.Message.FromWxID,
		s.Message.SenderWxID,
		s.Message.IsAtMe,
		s.Message.AppMsgType,
		contentPreview,
	)
}

func (s *ChatRoomSettingsService) IsAITrigger() bool {
	s.isFreeReply = false
	messageContent := s.Message.Content
	if s.Message.AppMsgType == model.AppMsgTypequote {
		var xmlMessage robot.XmlMessage
		if err := vars.RobotRuntime.XmlDecoder(messageContent, &xmlMessage); err == nil {
			messageContent = xmlMessage.AppMsg.Title
		}
	}
	if s.Message.IsAtMe {
		// 是否是 @所有人
		atAllRegex := regexp.MustCompile(vars.AtAllRegexp)
		if atAllRegex.MatchString(messageContent) {
			// 如果是 @所有人，则不处理
			return false
		}
		s.logAITrigger("mentioned", "", messageContent)
		return true
	}
	if matched, reason, triggerWord := s.matchConfiguredAITrigger(messageContent); matched {
		s.logAITrigger(reason, triggerWord, messageContent)
		return true
	}
	if s.shouldFreeReply(messageContent) {
		s.isFreeReply = true
		s.logAITrigger("free_reply", "", messageContent)
		return true
	}
	return false
}

func (s *ChatRoomSettingsService) matchConfiguredAITrigger(messageContent string) (bool, string, string) {
	if s.chatRoomSettings == nil {
		if s.globalSettings == nil {
			return false, "", ""
		}
		if s.globalSettings.ChatAIEnabled == nil || !*s.globalSettings.ChatAIEnabled {
			return false, "", ""
		}
		if s.globalSettings.ChatAITrigger != nil && *s.globalSettings.ChatAITrigger != "" && strings.HasPrefix(messageContent, *s.globalSettings.ChatAITrigger) {
			return true, "trigger_word.global", *s.globalSettings.ChatAITrigger
		}
		return false, "", ""
	}
	if s.chatRoomSettings.ChatAIEnabled == nil || !*s.chatRoomSettings.ChatAIEnabled {
		return false, "", ""
	}
	if s.chatRoomSettings.ChatAITrigger != nil && *s.chatRoomSettings.ChatAITrigger != "" {
		if strings.HasPrefix(messageContent, *s.chatRoomSettings.ChatAITrigger) {
			return true, "trigger_word.chat_room", *s.chatRoomSettings.ChatAITrigger
		}
		return false, "", ""
	}
	if s.globalSettings != nil && s.globalSettings.ChatAITrigger != nil && *s.globalSettings.ChatAITrigger != "" &&
		strings.HasPrefix(messageContent, *s.globalSettings.ChatAITrigger) {
		return true, "trigger_word.global_fallback", *s.globalSettings.ChatAITrigger
	}
	return false, "", ""
}

func (s *ChatRoomSettingsService) IsFreeReply() bool {
	return s.isFreeReply
}

func (s *ChatRoomSettingsService) GetAITriggerWord() string {
	if s.chatRoomSettings != nil && s.chatRoomSettings.ChatAITrigger != nil && *s.chatRoomSettings.ChatAITrigger != "" {
		return *s.chatRoomSettings.ChatAITrigger
	}
	if s.globalSettings != nil && s.globalSettings.ChatAITrigger != nil && *s.globalSettings.ChatAITrigger != "" {
		return *s.globalSettings.ChatAITrigger
	}
	return ""
}

func (s *ChatRoomSettingsService) GetChatRoomWelcomeConfig(chatRoomID string) (*model.ChatRoomSettings, error) {
	globalSettings, err := s.gsRepo.GetGlobalSettings()
	if err != nil {
		return nil, err
	}
	if globalSettings == nil {
		return nil, fmt.Errorf("加载全局配置失败")
	}
	chatRoomSetting, err := s.crsRepo.GetChatRoomSettings(chatRoomID)
	if err != nil {
		return nil, err
	}
	if chatRoomSetting == nil {
		return &model.ChatRoomSettings{
			WelcomeEnabled:  globalSettings.WelcomeEnabled,
			WelcomeType:     globalSettings.WelcomeType,
			WelcomeText:     globalSettings.WelcomeText,
			WelcomeEmojiMD5: globalSettings.WelcomeEmojiMD5,
			WelcomeEmojiLen: globalSettings.WelcomeEmojiLen,
			WelcomeImageURL: globalSettings.WelcomeImageURL,
			WelcomeURL:      globalSettings.WelcomeURL,
		}, nil
	}
	return chatRoomSetting, nil
}

func (s *ChatRoomSettingsService) GetPatConfig() settings.PatConfig {
	if s.chatRoomSettings != nil {
		if s.chatRoomSettings.PatEnabled != nil {
			return settings.PatConfig{
				PatEnabled:     *s.chatRoomSettings.PatEnabled,
				PatType:        s.chatRoomSettings.PatType,
				PatText:        s.chatRoomSettings.PatText,
				PatVoiceTimbre: s.chatRoomSettings.PatVoiceTimbre,
			}
		}
	}
	if s.globalSettings != nil {
		if s.globalSettings.PatEnabled != nil {
			return settings.PatConfig{
				PatEnabled:     *s.globalSettings.PatEnabled,
				PatType:        s.globalSettings.PatType,
				PatText:        s.globalSettings.PatText,
				PatVoiceTimbre: s.globalSettings.PatVoiceTimbre,
			}
		}
	}
	return settings.PatConfig{}
}

func (s *ChatRoomSettingsService) GetLeaveChatRoomConfig(chatRoomID string) *model.ChatRoomSettings {
	globalSettings, err := s.gsRepo.GetGlobalSettings()
	if err != nil {
		return nil
	}
	chatRoomSettings, err := s.crsRepo.GetChatRoomSettings(chatRoomID)
	if err != nil {
		return nil
	}
	if chatRoomSettings != nil {
		return chatRoomSettings
	}
	if globalSettings != nil {
		return &model.ChatRoomSettings{
			LeaveChatRoomAlertEnabled: globalSettings.LeaveChatRoomAlertEnabled,
			LeaveChatRoomAlertText:    globalSettings.LeaveChatRoomAlertText,
		}
	}
	return nil
}

func (s *ChatRoomSettingsService) GetAllEnableChatRank() ([]*model.ChatRoomSettings, error) {
	if vars.RobotRuntime.Status == model.RobotStatusOffline {
		return []*model.ChatRoomSettings{}, nil
	}
	return s.crsRepo.GetAllEnableChatRank()
}

func (s *ChatRoomSettingsService) GetAllEnableAISummary() ([]*model.ChatRoomSettings, error) {
	if vars.RobotRuntime.Status == model.RobotStatusOffline {
		return []*model.ChatRoomSettings{}, nil
	}
	return s.crsRepo.GetAllEnableAISummary()
}

func (s *ChatRoomSettingsService) GetAllEnableGoodMorning() ([]*model.ChatRoomSettings, error) {
	if vars.RobotRuntime.Status == model.RobotStatusOffline {
		return []*model.ChatRoomSettings{}, nil
	}
	return s.crsRepo.GetAllEnableGoodMorning()
}

func (s *ChatRoomSettingsService) GetAllEnableNews() ([]*model.ChatRoomSettings, error) {
	if vars.RobotRuntime.Status == model.RobotStatusOffline {
		return []*model.ChatRoomSettings{}, nil
	}
	return s.crsRepo.GetAllEnableNews()
}

func (s *ChatRoomSettingsService) SaveChatRoomSettings(data *model.ChatRoomSettings) error {
	if data == nil || strings.TrimSpace(data.ChatRoomID) == "" {
		return errors.New("群聊 ID 不能为空")
	}
	if err := s.validateAIProviderSelections(data); err != nil {
		return err
	}
	if err := validateFreeReplySettings(data); err != nil {
		return err
	}
	if err := s.normalizeKnowledgeCategories(data); err != nil {
		return err
	}
	if err := s.normalizeMemoryExtractionBlacklist(data); err != nil {
		return err
	}
	return s.crsRepo.SaveByChatRoomID(data)
}

func (s *ChatRoomSettingsService) validateAIProviderSelections(data *model.ChatRoomSettings) error {
	if data == nil {
		return nil
	}
	global, err := s.gsRepo.GetGlobalSettings()
	if err != nil {
		return err
	}
	if global == nil {
		return errors.New("全局设置不存在")
	}
	var existing *model.ChatRoomSettings
	if data.ChatRoomID != "" {
		existing, err = s.crsRepo.GetChatRoomSettings(data.ChatRoomID)
		if err != nil {
			return err
		}
	}
	providerSelection := func(requested *int64, existingValue func(*model.ChatRoomSettings) *int64) *int64 {
		if requested != nil {
			return requested
		}
		if existing != nil {
			return existingValue(existing)
		}
		return nil
	}
	modelSelection := func(requested *string, existingValue func(*model.ChatRoomSettings) *string, globalValue string) string {
		if requested != nil {
			if value := strings.TrimSpace(*requested); value != "" {
				return value
			}
			return strings.TrimSpace(globalValue)
		}
		if existing != nil {
			if value := existingValue(existing); value != nil && strings.TrimSpace(*value) != "" {
				return strings.TrimSpace(*value)
			}
		}
		return strings.TrimSpace(globalValue)
	}
	legacyProviderID := providerSelection(data.AIProviderID, func(row *model.ChatRoomSettings) *int64 { return row.AIProviderID })
	chatOverrideID := providerSelection(data.ChatAIProviderID, func(row *model.ChatRoomSettings) *int64 { return row.ChatAIProviderID })
	visionOverrideID := providerSelection(data.ImageRecognitionProviderID, func(row *model.ChatRoomSettings) *int64 {
		return row.ImageRecognitionProviderID
	})
	imageOverrideID := providerSelection(data.ImageGenerationProviderID, func(row *model.ChatRoomSettings) *int64 {
		return row.ImageGenerationProviderID
	})
	chatProviderID, visionProviderID, imageProviderID := resolveAIProviderIDsForTarget(
		global, legacyProviderID, chatOverrideID, visionOverrideID, imageOverrideID,
	)
	summaryProviderID := providerIDOrFallback(global.SummaryAIProviderID, global.AIProviderID)
	summaryOverrideID := providerSelection(data.SummaryAIProviderID, func(row *model.ChatRoomSettings) *int64 {
		return row.SummaryAIProviderID
	})
	if summaryOverrideID != nil && *summaryOverrideID > 0 {
		summaryProviderID = summaryOverrideID
	}
	globalImageModel := ""
	if len(global.ImageAISettings) > 0 {
		imageSettings := map[string]any{}
		_ = json.Unmarshal(global.ImageAISettings, &imageSettings)
		globalImageModel, _ = imageSettings["model"].(string)
	}
	selections := []struct {
		label              string
		providerID         *int64
		modelName          string
		explicitProviderID *int64
		explicitModelName  *string
		setProviderID      func(*int64)
	}{
		{label: "AI回复", providerID: chatProviderID, modelName: modelSelection(data.ChatModel, func(row *model.ChatRoomSettings) *string { return row.ChatModel }, global.ChatModel), explicitProviderID: data.ChatAIProviderID, explicitModelName: data.ChatModel, setProviderID: func(id *int64) { data.ChatAIProviderID = id }},
		{label: "图像识别", providerID: visionProviderID, modelName: modelSelection(data.ImageRecognitionModel, func(row *model.ChatRoomSettings) *string { return row.ImageRecognitionModel }, global.ImageRecognitionModel), explicitProviderID: data.ImageRecognitionProviderID, explicitModelName: data.ImageRecognitionModel, setProviderID: func(id *int64) { data.ImageRecognitionProviderID = id }},
		{label: "AI绘图", providerID: imageProviderID, modelName: modelSelection(data.ImageGenerationModel, func(row *model.ChatRoomSettings) *string { return row.ImageGenerationModel }, globalImageModel), explicitProviderID: data.ImageGenerationProviderID, explicitModelName: data.ImageGenerationModel, setProviderID: func(id *int64) { data.ImageGenerationProviderID = id }},
		{label: "群聊总结", providerID: summaryProviderID, modelName: modelSelection(data.ChatRoomSummaryModel, func(row *model.ChatRoomSettings) *string { return row.ChatRoomSummaryModel }, global.ChatRoomSummaryModel), explicitProviderID: data.SummaryAIProviderID, explicitModelName: data.ChatRoomSummaryModel, setProviderID: func(id *int64) { data.SummaryAIProviderID = id }},
	}
	for _, selection := range selections {
		if selection.explicitProviderID != nil && *selection.explicitProviderID > 0 &&
			(selection.explicitModelName == nil || strings.TrimSpace(*selection.explicitModelName) == "") {
			return fmt.Errorf("选择%s渠道后必须选择模型", selection.label)
		}
		if selection.providerID == nil || *selection.providerID <= 0 || selection.modelName == "" {
			continue
		}
		provider, err := NewAIProviderService(s.ctx).GetEnabledByID(selection.providerID)
		if err != nil || provider == nil {
			return fmt.Errorf("%s渠道不存在或已停用", selection.label)
		}
		if validationErr := ValidateAIProviderModel(provider, selection.modelName, selection.label); validationErr != nil {
			matchedProvider, matchErr := NewAIProviderService(s.ctx).FindUniqueEnabledProviderForModel(selection.modelName)
			if matchErr != nil {
				return matchErr
			}
			if matchedProvider == nil {
				return validationErr
			}
			matchedProviderID := matchedProvider.ID
			selection.setProviderID(&matchedProviderID)
			log.Printf("[AIProviderRepair] room=%s capability=%s model=%s provider=%s", data.ChatRoomID, selection.label, selection.modelName, matchedProvider.Name)
		}
	}
	return nil
}

func validateFreeReplySettings(data *model.ChatRoomSettings) error {
	if data == nil || data.FreeReplyEnabled == nil || !*data.FreeReplyEnabled {
		return nil
	}
	if data.FreeReplyLevel == nil {
		return errors.New("自由回复参与频率不能为空")
	}
	level := strings.ToLower(strings.TrimSpace(*data.FreeReplyLevel))
	if level != "active" && level != "normal" && level != "cautious" && level != "crazy" {
		return errors.New("自由回复参与频率参数错误")
	}
	data.FreeReplyLevel = &level
	if data.FreeReplyCooldownSeconds == nil || *data.FreeReplyCooldownSeconds < 0 {
		return errors.New("自由回复冷却时间不能小于0")
	}
	if data.FreeReplyDailyLimit == nil || *data.FreeReplyDailyLimit < 0 {
		return errors.New("自由回复每日上限不能小于0")
	}
	return nil
}

func normalizeKnowledgeCategoryCodes(codes []string) []string {
	return normalizeStringList(codes)
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	return normalized
}

func (s *ChatRoomSettingsService) normalizeKnowledgeCategories(data *model.ChatRoomSettings) error {
	if data == nil || data.KnowledgeCategories == nil {
		return nil
	}

	codes, err := data.GetKnowledgeCategoryCodes()
	if err != nil {
		return fmt.Errorf("knowledge_categories 格式错误: %w", err)
	}

	normalized := normalizeKnowledgeCategoryCodes(codes)
	payload, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("序列化 knowledge_categories 失败: %w", err)
	}
	data.KnowledgeCategories = payload

	if len(normalized) == 0 {
		return nil
	}

	categoryRepo := repository.NewKnowledgeCategoryRepo(s.ctx, vars.DB)
	categories, err := categoryRepo.GetByCodes(normalized)
	if err != nil {
		return fmt.Errorf("查询知识库分类失败: %w", err)
	}

	categoryByCode := make(map[string]*model.KnowledgeCategory, len(categories))
	for _, category := range categories {
		categoryByCode[category.Code] = category
	}

	missingCodes := make([]string, 0)
	unsupportedCodes := make([]string, 0)
	for _, code := range normalized {
		category, ok := categoryByCode[code]
		if !ok {
			missingCodes = append(missingCodes, code)
			continue
		}
		if category.Type != model.KnowledgeCategoryTypeText {
			unsupportedCodes = append(unsupportedCodes, code)
		}
	}

	if len(missingCodes) > 0 {
		return fmt.Errorf("以下知识库不存在: %s", strings.Join(missingCodes, ", "))
	}
	if len(unsupportedCodes) > 0 {
		return fmt.Errorf("以下知识库不是文本知识库，暂不支持绑定到群聊: %s", strings.Join(unsupportedCodes, ", "))
	}

	return nil
}

func (s *ChatRoomSettingsService) normalizeMemoryExtractionBlacklist(data *model.ChatRoomSettings) error {
	if data == nil || data.MemoryExtractionBlacklist == nil {
		return nil
	}

	wxIDs, err := data.GetMemoryExtractionBlacklist()
	if err != nil {
		return fmt.Errorf("memory_extraction_blacklist 格式错误: %w", err)
	}

	payload, err := json.Marshal(normalizeStringList(wxIDs))
	if err != nil {
		return fmt.Errorf("序列化 memory_extraction_blacklist 失败: %w", err)
	}
	data.MemoryExtractionBlacklist = payload

	return nil
}
