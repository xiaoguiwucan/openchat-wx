package startup

import (
	"github.com/xiaoguiwucan/openchat-wx/plugin"
	"github.com/xiaoguiwucan/openchat-wx/plugin/plugins"
	"github.com/xiaoguiwucan/openchat-wx/vars"
)

func RegisterMessagePlugin() {
	vars.MessagePlugin = plugin.NewMessagePlugin()
	// 群聊聊天插件
	vars.MessagePlugin.Register(plugins.NewChatRoomAIChatPlugin())
	vars.MessagePlugin.Register(plugins.NewChatRoomMemberBlacklistPlugin())
	vars.MessagePlugin.Register(plugins.NewSwitchChatModelPlugin())
	vars.MessagePlugin.Register(plugins.NewSliderAccessSecretPlugin())
	vars.MessagePlugin.Register(plugins.NewChatRoomWxhbNotifyPlugin())
	vars.MessagePlugin.Register(plugins.NewPodcastPlugin())
	vars.MessagePlugin.Register(plugins.NewKnowledgeBasePlugin())
	// 朋友聊天插件
	vars.MessagePlugin.Register(plugins.NewFriendAIChatPlugin())
	// 群聊拍一拍交互插件
	vars.MessagePlugin.Register(plugins.NewPatPlugin())
	// 抖音解析插件
	vars.MessagePlugin.Register(plugins.NewDouyinVideoParsePlugin())
	// B站视频解析插件
	vars.MessagePlugin.Register(plugins.NewBilibiliVideoParsePlugin())
	// 图片自动上传插件
	vars.MessagePlugin.Register(plugins.NewImageAutoUploadPlugin())
}
