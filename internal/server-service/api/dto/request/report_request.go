package request

type ReportRequest struct {
	StartDate string `json:"start_date" binding:"required,datetime=2006-01-02"`
	EndDate   string `json:"end_date" binding:"required,datetime=2006-01-02"`
	Email     string `json:"email" binding:"required,email"`
}
