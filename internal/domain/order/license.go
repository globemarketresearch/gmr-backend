package order

// LicenseTier is a purchasable report license. Prices MUST stay in sync with
// the Global edition prices in gmr/lib/license-tiers.ts — the checkout page
// shows those prices, and this table is what the customer is actually charged.
type LicenseTier struct {
	ID    string
	Name  string
	Price float64
}

var licenseTiers = map[string]LicenseTier{
	"single":    {ID: "single", Name: "Single User License", Price: 3499},
	"corporate": {ID: "corporate", Name: "Corporate User License", Price: 5449},
	"excel":     {ID: "excel", Name: "Excel License", Price: 1799},
	// "student" is intentionally absent: it requires university ID
	// verification, so it goes through the manual purchase-request flow.
}

// GetLicenseTier returns the online-purchasable license tier for id.
func GetLicenseTier(id string) (LicenseTier, bool) {
	t, ok := licenseTiers[id]
	return t, ok
}
