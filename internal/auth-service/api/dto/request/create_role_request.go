package request

type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	ScopeIDs    []string `json:"scope_ids"`
}
