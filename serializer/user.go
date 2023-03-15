package serializer

import "peer2http/db"

type User struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Status   string `json:"status"`
	Avatar   string `json:"avatar"`
	CreateAt int64  `json:"created_at"`
}

type UserResponse struct {
	Response
	Data User `json:"data"`
}

func BuildUser(user db.User) User {
	return User{
		ID:       user.ID,
		Username: user.Username,
		Status:   user.Status,
		Avatar:   user.Avatar,
		CreateAt: user.CreatedAt.Unix(),
	}
}

func BuildUserResponse(user db.User) UserResponse {
	return UserResponse{
		Data: BuildUser(user),
	}
}
