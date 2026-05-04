package paypal

// ============ Token ============

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"` // seconds
	TokenType   string `json:"token_type"`
}

// ============ Create Order ============

type CreateOrderRequest struct {
	Intent        string         `json:"intent"`
	PurchaseUnits []PurchaseUnit `json:"purchase_units"`
}

type PurchaseUnit struct {
	ReferenceID string `json:"reference_id,omitempty"`
	Description string `json:"description,omitempty"`
	Amount      Amount `json:"amount"`
}

type Amount struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

type CreateOrderResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Links  []Link `json:"links"`
}

type Link struct {
	Href   string `json:"href"`
	Rel    string `json:"rel"`
	Method string `json:"method"`
}

// ============ Capture Order ============

type CaptureOrderResponse struct {
	ID            string         `json:"id"`
	Status        string         `json:"status"`
	PurchaseUnits []PurchaseUnit `json:"purchase_units"`
	PaymentSource *PaymentSource `json:"payment_source,omitempty"`
	Captures      []Capture      `json:"-"` // flattened from PurchaseUnits
}

type PaymentSource struct {
	PayPal *PayPalAccount `json:"paypal,omitempty"`
	Card   *CardSource    `json:"card,omitempty"`
}

type PayPalAccount struct {
	EmailAddress string `json:"email_address,omitempty"`
	Name         *Name  `json:"name,omitempty"`
}

type CardSource struct {
	Name string `json:"name,omitempty"`
}

type Name struct {
	GivenName string `json:"given_name,omitempty"`
	Surname   string `json:"surname,omitempty"`
}

type PurchaseUnitCapture struct {
	Payments *PurchaseUnitPayments `json:"payments,omitempty"`
}

type PurchaseUnitPayments struct {
	Captures []Capture `json:"captures,omitempty"`
}

type Capture struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Amount Amount `json:"amount"`
}

// ============ Error ============

type ErrorResponse struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Details []struct {
		Field       string `json:"field"`
		Value       string `json:"value"`
		Location    string `json:"location"`
		Issue       string `json:"issue"`
		Description string `json:"description"`
	} `json:"details,omitempty"`
}
