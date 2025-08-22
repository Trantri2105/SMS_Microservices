package response

type UserInfoResponse struct {
	ID        string              `json:"id"`
	Email     string              `json:"email"`
	FirstName string              `json:"first_name"`
	LastName  string              `json:"last_name"`
	Roles     []RoleInfoResponse  `json:"roles,omitempty"`
	Scopes    []ScopeInfoResponse `json:"scopes,omitempty"`
}
