package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/synctv-org/synctv/internal/db"
	dbModel "github.com/synctv-org/synctv/internal/model"
	"github.com/synctv-org/synctv/internal/op"
	"github.com/synctv-org/synctv/internal/settings"
	"github.com/synctv-org/synctv/server/model"
	"gorm.io/gorm"
)

func EditAdminSettings(ctx *gin.Context) {
	// user := ctx.MustGet("user").(*op.User)

	req := model.AdminSettingsReq{}
	if err := model.Decode(ctx, &req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
		return
	}

	for k, v := range req {
		err := settings.SetValue(k, v)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
			return
		}
	}

	ctx.Status(http.StatusNoContent)
}

func AdminSettings(ctx *gin.Context) {
	// user := ctx.MustGet("user").(*op.User)
	group := ctx.Param("group")
	if group == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("group is required"))
		return
	}

	s, ok := settings.GroupSettings[dbModel.SettingGroup(group)]
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("group not found"))
		return
	}
	resp := make(gin.H, len(s))
	for _, v := range s {
		resp[v.Name()] = v.Interface()
	}

	ctx.JSON(http.StatusOK, model.NewApiDataResp(resp))
}

func Users(ctx *gin.Context) {
	// user := ctx.MustGet("user").(*op.User)
	order := ctx.Query("order")
	if order == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("order is required"))
		return
	}

	page, pageSize, err := GetPageAndPageSize(ctx)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
		return
	}

	var desc = ctx.DefaultQuery("sort", "desc") == "desc"

	scopes := []func(db *gorm.DB) *gorm.DB{}

	if keyword := ctx.Query("keyword"); keyword != "" {
		scopes = append(scopes, db.WhereUserNameLike(keyword))
	}

	switch order {
	case "createdAt":
		if desc {
			scopes = append(scopes, db.OrderByCreatedAtDesc)
		} else {
			scopes = append(scopes, db.OrderByCreatedAtAsc)
		}
	case "name":
		if desc {
			scopes = append(scopes, db.OrderByDesc("username"))
		} else {
			scopes = append(scopes, db.OrderByAsc("username"))
		}
	case "id":
		if desc {
			scopes = append(scopes, db.OrderByIDDesc)
		} else {
			scopes = append(scopes, db.OrderByIDAsc)
		}
	default:
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("not support order"))
		return
	}

	ctx.JSON(http.StatusOK, model.NewApiDataResp(gin.H{
		"total": db.GetAllUserCountWithRole(dbModel.RoleUser, scopes...),
		"list":  genUserListResp(dbModel.RoleUser, append(scopes, db.Paginate(page, pageSize))...),
	}))
}

func genUserListResp(role dbModel.Role, scopes ...func(db *gorm.DB) *gorm.DB) []*model.UserInfoResp {
	us := db.GetAllUserWithRoleUser(role, scopes...)
	resp := make([]*model.UserInfoResp, len(us))
	for i, v := range us {
		resp[i] = &model.UserInfoResp{
			ID:        v.ID,
			Username:  v.Username,
			Role:      v.Role,
			CreatedAt: v.CreatedAt.UnixMilli(),
		}
	}
	return resp
}

func PendingUsers(ctx *gin.Context) {
	// user := ctx.MustGet("user").(*op.User)
	order := ctx.Query("order")
	if order == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("order is required"))
		return
	}

	page, pageSize, err := GetPageAndPageSize(ctx)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
		return
	}

	var desc = ctx.DefaultQuery("sort", "desc") == "desc"

	scopes := []func(db *gorm.DB) *gorm.DB{}

	if keyword := ctx.Query("keyword"); keyword != "" {
		scopes = append(scopes, db.WhereUserNameLike(keyword))
	}

	switch order {
	case "createdAt":
		if desc {
			scopes = append(scopes, db.OrderByCreatedAtDesc)
		} else {
			scopes = append(scopes, db.OrderByCreatedAtAsc)
		}
	case "name":
		if desc {
			scopes = append(scopes, db.OrderByDesc("username"))
		} else {
			scopes = append(scopes, db.OrderByAsc("username"))
		}
	case "id":
		if desc {
			scopes = append(scopes, db.OrderByIDDesc)
		} else {
			scopes = append(scopes, db.OrderByIDAsc)
		}
	default:
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("not support order"))
		return
	}

	ctx.JSON(http.StatusOK, model.NewApiDataResp(gin.H{
		"total": db.GetAllUserCountWithRole(dbModel.RolePending, scopes...),
		"list":  genUserListResp(dbModel.RolePending, append(scopes, db.Paginate(page, pageSize))...),
	}))
}

func ApprovePendingUser(ctx *gin.Context) {
	req := model.UserIDReq{}
	if err := model.Decode(ctx, &req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
		return
	}

	user, err := db.GetUserByID(req.ID)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
		return
	}

	if !user.IsPending() {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("user is not pending"))
		return
	}

	err = db.SetRoleByID(req.ID, dbModel.RoleUser)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, model.NewApiErrorResp(err))
		return
	}

	ctx.Status(http.StatusNoContent)
}

func BanUser(ctx *gin.Context) {
	user := ctx.MustGet("user").(*op.User)

	req := model.UserIDReq{}
	if err := model.Decode(ctx, &req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
		return
	}

	u, err := op.GetUserById(req.ID)
	if err != nil {
		if errors.Is(err, op.ErrUserPending) {
			err = db.SetRoleByID(req.ID, dbModel.RoleBanned)
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, model.NewApiErrorResp(err))
				return
			}
		} else {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
		}
		return
	}

	if u.ID == user.ID {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("cannot ban yourself"))
		return
	}
	if u.IsRoot() {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("cannot ban root user"))
		return
	}

	err = u.SetRole(dbModel.RoleBanned)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, model.NewApiErrorResp(err))
		return
	}

	ctx.Status(http.StatusNoContent)
}

func PendingRooms(ctx *gin.Context) {
	// user := ctx.MustGet("user").(*op.User)
	order := ctx.Query("order")
	if order == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("order is required"))
		return
	}

	page, pageSize, err := GetPageAndPageSize(ctx)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
		return
	}

	var desc = ctx.DefaultQuery("sort", "desc") == "desc"

	scopes := []func(db *gorm.DB) *gorm.DB{
		db.WhereStatus(dbModel.RoomStatusPending),
	}

	if keyword := ctx.Query("keyword"); keyword != "" {
		scopes = append(scopes, db.WhereRoomNameLike(keyword))
	}

	switch order {
	case "createdAt":
		if desc {
			scopes = append(scopes, db.OrderByCreatedAtDesc)
		} else {
			scopes = append(scopes, db.OrderByCreatedAtAsc)
		}
	case "name":
		if desc {
			scopes = append(scopes, db.OrderByDesc("name"))
		} else {
			scopes = append(scopes, db.OrderByAsc("name"))
		}
	case "id":
		if desc {
			scopes = append(scopes, db.OrderByIDDesc)
		} else {
			scopes = append(scopes, db.OrderByIDAsc)
		}
	default:
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("not support order"))
		return
	}

	if keyword := ctx.Query("keyword"); keyword != "" {
		// search mode, all, name, creator
		switch ctx.DefaultQuery("search", "all") {
		case "all":
			scopes = append(scopes, db.WhereRoomNameLikeOrCreatorIn(keyword, db.GerUsersIDByUsernameLike(keyword)))
		case "name":
			scopes = append(scopes, db.WhereRoomNameLike(keyword))
		case "creator":
			scopes = append(scopes, db.WhereCreatorIDIn(db.GerUsersIDByUsernameLike(keyword)))
		}
	}

	ctx.JSON(http.StatusOK, model.NewApiDataResp(gin.H{
		"total": db.GetAllRoomsWithoutHiddenCount(scopes...),
		"list":  genRoomListResp(append(scopes, db.Paginate(page, pageSize))...),
	}))
}

func ApprovePendingRoom(ctx *gin.Context) {
	req := model.RoomIDReq{}
	if err := model.Decode(ctx, &req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
		return
	}

	room, err := db.GetRoomByID(req.Id)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
		return
	}

	if !room.IsPending() {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("room is not pending"))
		return
	}

	err = db.SetRoomStatus(req.Id, dbModel.RoomStatusActive)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, model.NewApiErrorResp(err))
		return
	}

	ctx.Status(http.StatusNoContent)
}

func BanRoom(ctx *gin.Context) {
	user := ctx.MustGet("user").(*op.User)

	req := model.RoomIDReq{}
	if err := model.Decode(ctx, &req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
		return
	}

	room, err := op.LoadOrInitRoomByID(req.Id)
	if err != nil {
		if errors.Is(err, op.ErrRoomPending) || errors.Is(err, op.ErrRoomStopped) {
			err = db.SetRoomStatus(req.Id, dbModel.RoomStatusBanned)
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, model.NewApiErrorResp(err))
				return
			}
		} else {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
		}
		return
	}

	creator, err := db.GetUserByID(room.CreatorID)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorResp(err))
		return
	}

	if creator.ID == user.ID {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("cannot ban yourself"))
		return
	}

	if creator.IsAdmin() {
		ctx.AbortWithStatusJSON(http.StatusForbidden, model.NewApiErrorStringResp("no permission"))
		return
	}

	if room.IsBanned() {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, model.NewApiErrorStringResp("room is already banned"))
		return
	}

	err = room.SetRoomStatus(dbModel.RoomStatusBanned)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, model.NewApiErrorResp(err))
		return
	}

	ctx.Status(http.StatusNoContent)
}