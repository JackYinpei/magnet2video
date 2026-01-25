// Package controller provides user controller
// Author: Done-0
// Created: 2026-01-22
package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Done-0/gin-scaffold/internal/types/errno"
	"github.com/Done-0/gin-scaffold/internal/utils/errorx"
	"github.com/Done-0/gin-scaffold/internal/utils/validator"
	"github.com/Done-0/gin-scaffold/internal/utils/vo"
	"github.com/Done-0/gin-scaffold/pkg/serve/controller/dto"
	"github.com/Done-0/gin-scaffold/pkg/serve/service"
)

// UserController user HTTP controller
type UserController struct {
	userService service.UserService
}

// NewUserController creates user controller
func NewUserController(userService service.UserService) *UserController {
	return &UserController{
		userService: userService,
	}
}

// Register handles user registration
// @Router /api/v1/auth/register [post]
func (uc *UserController) Register(c *gin.Context) {
	req := &dto.RegisterRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	errors := validator.Validate(req)
	if errors != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, errors, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "validation failed"))))
		return
	}

	response, err := uc.userService.Register(c, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrUserAlreadyExists, errorx.KV("email", req.Email))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// Login handles user login
// @Router /api/v1/auth/login [post]
func (uc *UserController) Login(c *gin.Context) {
	req := &dto.LoginRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	errors := validator.Validate(req)
	if errors != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, errors, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "validation failed"))))
		return
	}

	response, err := uc.userService.Login(c, req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidCredentials)))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// GetProfile handles getting current user's profile
// @Router /api/v1/user/profile [get]
func (uc *UserController) GetProfile(c *gin.Context) {
	response, err := uc.userService.GetProfile(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrUserNotFound, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// UpdateProfile handles updating current user's profile
// @Router /api/v1/user/profile [put]
func (uc *UserController) UpdateProfile(c *gin.Context) {
	req := &dto.UpdateProfileRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	response, err := uc.userService.UpdateProfile(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// ChangePassword handles password change
// @Router /api/v1/user/password [put]
func (uc *UserController) ChangePassword(c *gin.Context) {
	req := &dto.ChangePasswordRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	errors := validator.Validate(req)
	if errors != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, errors, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "validation failed"))))
		return
	}

	response, err := uc.userService.ChangePassword(c, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidCredentials)))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// SetTorrentPublic handles setting torrent visibility
// @Router /api/v1/user/torrent/public [post]
func (uc *UserController) SetTorrentPublic(c *gin.Context) {
	req := &dto.SetTorrentPublicRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	errors := validator.Validate(req)
	if errors != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, errors, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "validation failed"))))
		return
	}

	response, err := uc.userService.SetTorrentPublic(c, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrResourceNotFound, errorx.KV("resource", "torrent"), errorx.KV("id", req.InfoHash))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}
