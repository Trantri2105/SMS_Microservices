package response

type UserInfoResponse struct {
	ID        string              `json:"id"`
	Email     string              `json:"email"`
	FirstName string              `json:"first_name,omitempty"`
	LastName  string              `json:"last_name,omitempty"`
	Roles     []RoleInfoResponse  `json:"roles,omitempty"`
	Scopes    []ScopeInfoResponse `json:"scopes,omitempty"`
}
