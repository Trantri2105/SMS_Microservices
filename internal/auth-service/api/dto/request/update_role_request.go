package request

type UpdateRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ScopeIDs    []string `json:"scope_ids" binding:"omitempty,dive,uuid"`
}
