package domain

type User struct {
	UserID       string `json:"userId"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	LastName     string `json:"lastName"`
	State        string `json:"state"`
	RegisteredAt string `json:"registeredAt"`
}
