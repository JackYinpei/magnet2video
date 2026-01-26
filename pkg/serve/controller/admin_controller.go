// Package controller provides admin controller
// Author: Done-0
// Created: 2026-01-26
package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Done-0/gin-scaffold/internal/types/errno"
	"github.com/Done-0/gin-scaffold/internal/utils/errorx"
	"github.com/Done-0/gin-scaffold/internal/utils/validator"
	"github.com/Done-0/gin-scaffold/internal/utils/vo"
	"github.com/Done-0/gin-scaffold/pkg/serve/controller/dto"
	"github.com/Done-0/gin-scaffold/pkg/serve/service"
)

// AdminController admin HTTP controller
type AdminController struct {
	adminService service.AdminService
}

// NewAdminController creates admin controller
func NewAdminController(adminService service.AdminService) *AdminController {
	return &AdminController{
		adminService: adminService,
	}
}

// ListUsers handles listing all users
// @Router /api/v1/admin/users [get]
func (ac *AdminController) ListUsers(c *gin.Context) {
	req := &dto.ListUsersRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind query failed"))))
		return
	}

	response, err := ac.adminService.ListUsers(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// GetUserDetail handles getting user detail
// @Router /api/v1/admin/users/:id [get]
func (ac *AdminController) GetUserDetail(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "invalid user ID"))))
		return
	}

	response, err := ac.adminService.GetUserDetail(c, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrUserNotFound, errorx.KV("id", userIDStr))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// GetUserTorrents handles getting user's torrents
// @Router /api/v1/admin/users/:id/torrents [get]
func (ac *AdminController) GetUserTorrents(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "invalid user ID"))))
		return
	}

	response, err := ac.adminService.GetUserTorrents(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// DeleteUser handles deleting a user
// @Router /api/v1/admin/users/:id [delete]
func (ac *AdminController) DeleteUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "invalid user ID"))))
		return
	}

	response, err := ac.adminService.DeleteUser(c, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// UpdateUserRole handles updating a user's role
// @Router /api/v1/admin/users/:id/role [put]
func (ac *AdminController) UpdateUserRole(c *gin.Context) {
	req := &dto.UpdateUserRoleRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	// Get user ID from path parameter
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "invalid user ID"))))
		return
	}
	req.UserID = userID

	errors := validator.Validate(req)
	if errors != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, errors, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "validation failed"))))
		return
	}

	response, err := ac.adminService.UpdateUserRole(c, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// ListAllTorrents handles listing all torrents
// @Router /api/v1/admin/torrents [get]
func (ac *AdminController) ListAllTorrents(c *gin.Context) {
	req := &dto.ListAllTorrentsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind query failed"))))
		return
	}

	response, err := ac.adminService.ListAllTorrents(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// DeleteTorrent handles deleting a torrent
// @Router /api/v1/admin/torrents/:info_hash [delete]
func (ac *AdminController) DeleteTorrent(c *gin.Context) {
	infoHash := c.Param("info_hash")
	if infoHash == "" {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "missing info_hash"))))
		return
	}

	response, err := ac.adminService.DeleteTorrent(c, infoHash)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrTorrentNotFound, errorx.KV("info_hash", infoHash))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// GetStats handles getting system statistics
// @Router /api/v1/admin/stats [get]
func (ac *AdminController) GetStats(c *gin.Context) {
	response, err := ac.adminService.GetStats(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}
