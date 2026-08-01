package controller

import (
	"errors"
	"time"
	"unicode/utf8"

	"github.com/xiaoguiwucan/openchat-wx/dto"
	"github.com/xiaoguiwucan/openchat-wx/model"
	"github.com/xiaoguiwucan/openchat-wx/pkg/appx"
	"github.com/xiaoguiwucan/openchat-wx/service"

	"github.com/gin-gonic/gin"
)

type ChatRoom struct {
}

func NewChatRoomController() *ChatRoom {
	return &ChatRoom{}
}

func (cr *ChatRoom) SyncChatRoomMember(c *gin.Context) {
	var req dto.SyncChatRoomMemberRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	service.NewChatRoomService(c).SyncChatRoomMember(req.ChatRoomID)
	resp.ToResponse(nil)
}

func (cr *ChatRoom) RunChatRoomSummary(c *gin.Context) {
	var req dto.RunChatRoomSummaryRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	if req.EndTime == 0 {
		req.EndTime = time.Now().Unix()
	}
	if req.StartTime == 0 {
		req.StartTime = req.EndTime - int64((24 * time.Hour).Seconds())
	}
	if req.StartTime >= req.EndTime || req.EndTime-req.StartTime > int64((7*24*time.Hour).Seconds()) {
		resp.ToErrorResponse(errors.New("总结时间范围必须在 7 天内，且开始时间早于结束时间"))
		return
	}
	if err := service.NewChatRoomService(c).ChatRoomAISummaryByRange(req.ChatRoomID, req.StartTime, req.EndTime); err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}

func (cr *ChatRoom) GetChatRoomMembers(c *gin.Context) {
	var req dto.ChatRoomMemberListRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	pager := appx.InitPager(c)
	list, total, err := service.NewChatRoomService(c).GetChatRoomMembers(req, pager)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponseList(list, total)
}

func (cr *ChatRoom) GetNotLeftMembers(c *gin.Context) {
	var req dto.ChatRoomMemberListRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	list, err := service.NewChatRoomService(c).GetNotLeftMembers(req)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(list)
}

func (cr *ChatRoom) GetChatRoomMember(c *gin.Context) {
	var req dto.ChatRoomMemberRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	member, err := service.NewChatRoomService(c).GetChatRoomMember(req)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(member)
}

func (cr *ChatRoom) UpdateChatRoomMember(c *gin.Context) {
	var req model.UpdateChatRoomMember
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	var err error
	if req.Batch {
		err = service.NewChatRoomService(c).BatchUpdateChatRoomMemberInfo(req)
	} else {
		err = service.NewChatRoomService(c).UpdateChatRoomMemberInfo(req)
	}
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}

func (cr *ChatRoom) CreateChatRoom(c *gin.Context) {
	var req dto.CreateChatRoomRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	err := service.NewChatRoomService(c).CreateChatRoom(req.ContactIDs)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}

func (cr *ChatRoom) InviteChatRoomMember(c *gin.Context) {
	var req dto.InviteChatRoomMemberRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	err := service.NewChatRoomService(c).InviteChatRoomMember(req.ChatRoomID, req.ContactIDs)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}

func (cr *ChatRoom) GroupConsentToJoin(c *gin.Context) {
	var req dto.GroupConsentToJoinRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	err := service.NewChatRoomService(c).GroupConsentToJoin(req.SystemMessageID)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}

func (cr *ChatRoom) GroupSetChatRoomName(c *gin.Context) {
	var req dto.ChatRoomOperateRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}

	if utf8.RuneCountInString(req.Content) > 30 {
		resp.ToErrorResponse(errors.New("群名称不能超过30个字符"))
		return
	}
	err := service.NewChatRoomService(c).GroupSetChatRoomName(req.ChatRoomID, req.Content)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}

func (cr *ChatRoom) GroupSetChatRoomRemarks(c *gin.Context) {
	var req dto.ChatRoomOperateRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	if utf8.RuneCountInString(req.Content) > 30 {
		resp.ToErrorResponse(errors.New("群备注不能超过30个字符"))
		return
	}
	err := service.NewChatRoomService(c).GroupSetChatRoomRemarks(req.ChatRoomID, req.Content)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}

func (cr *ChatRoom) GroupSetChatRoomAnnouncement(c *gin.Context) {
	var req dto.ChatRoomOperateRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	err := service.NewChatRoomService(c).GroupSetChatRoomAnnouncement(req.ChatRoomID, req.Content)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}

func (cr *ChatRoom) GroupDelChatRoomMember(c *gin.Context) {
	var req dto.DelChatRoomMemberRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	err := service.NewChatRoomService(c).GroupDelChatRoomMember(req.ChatRoomID, req.MemberIDs)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}

func (cr *ChatRoom) GroupQuit(c *gin.Context) {
	var req dto.ChatRoomRequestBase
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	err := service.NewChatRoomService(c).GroupQuit(req.ChatRoomID)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}
