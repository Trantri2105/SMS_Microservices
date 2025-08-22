package request

type UpdateUserInfoRequest struct {
	Email     string `json:"email" binding:"omitempty,email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
