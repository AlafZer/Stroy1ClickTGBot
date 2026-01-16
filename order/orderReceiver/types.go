package order

import "time"

type Order struct {
	ID           int64       `json:"id"`
	Notes        string      `json:"notes"`
	OrderStatus  OrderStatus `json:"orderStatus"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
	OrderItems   []OrderItem `json:"orderItems"`
	ContactPhone string      `json:"contactPhone"`
	UserID       int64       `json:"userId"`
}

type OrderItem struct {
	ID        int64 `json:"id"`
	ProductID int   `json:"productId"`
	Quantity  int   `json:"quantity"`
}

type OrderStatus int

const (
	Created OrderStatus = iota
	Paid
	Shipped
	Delivered
	Canceled
)
