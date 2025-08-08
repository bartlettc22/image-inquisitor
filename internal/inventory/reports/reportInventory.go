package reports

import "github.com/bartlettc22/image-inquisitor/internal/inventory"

func ExportInventoryReport(inventory *inventory.Inventory) *Report {
	return wrapReport("inventory", inventory)
}

func GenerateInventoryReport(inventory *inventory.Inventory) *Report {
	return wrapReport("inventory", inventory)
}
