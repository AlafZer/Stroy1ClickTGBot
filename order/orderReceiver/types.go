package order

import "time"

type Order struct {
	ID              int64       `json:"id"`
	Notes           string      `json:"notes"`
	OrderStatus     OrderStatus `json:"order_status"`
	DeliveryAddress string      `json:"delivery_address"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	OrderItems      []OrderItem `json:"order_items"`
	ContactPhone    string      `json:"contact_phone"`
	LegalName       string      `json:"legal_name"`
	LegalForm       LegalForm   `json:"legal_form"`
	INN             string      `json:"inn"`
	KPP             string      `json:"kpp"`
	ContactName     string      `json:"contact_name"`
	ContactEmail    string      `json:"email"`
	UserID          int64       `json:"user_id"`
}

type OrderItem struct {
	ID           int64  `json:"id"`
	ProductID    int    `json:"product_id"`
	ProductTitle string `json:"product_title"`
	Price        int    `json:"price"`
	Quantity     int    `json:"quantity"`
	Unit         Unit   `json:"unit"`
}

type OrderStatus int

const (
	Created OrderStatus = iota
	Paid
	Shipped
	Delivered
	Canceled
)

type LegalForm int

const (
	LLC LegalForm = iota
	IE
)

type Unit int

const (
	PIECE Unit = iota // Штука
	KG                // Килограмм
	GRAM              // Грамм
	LITER             // Литр
	ML                // Миллилитр
	METER             // Метр
	PACK              // Упаковка
)
