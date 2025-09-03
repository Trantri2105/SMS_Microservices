package request

type UpdateUserRoleRequest struct {
	RoleIDs []string `json:"role_ids" binding:"required,dive,uuid"`
}
