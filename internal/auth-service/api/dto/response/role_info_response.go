package response

type RoleInfoResponse struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Scopes      []ScopeInfoResponse `json:"scopes,omitempty"`
}
