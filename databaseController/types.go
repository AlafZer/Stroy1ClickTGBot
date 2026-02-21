package dbController

type DatabaseQuerySetAdmin struct {
	UserID int64 `json:"user_id"`
	State  bool  `json:"statement"`
}
