package http

type LoginRequest struct {
	Password string `json:"password" binding:"required,min=8,max=256"`
	Email    string `json:"email" binding:"required,min=6,max=254"`
}

type RegisterRequest struct {
	Password string `json:"password" binding:"required,min=8,max=256"`
	Email    string `json:"email" binding:"required,min=6,max=254"`
}
