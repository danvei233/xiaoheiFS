package integration

import (
	"context"
	"sort"

	appshared "xiaoheiplay/internal/app/shared"
)

type AutomationCatalogLineOption struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	AreaID int64  `json:"area_id"`
	State  int    `json:"state"`
}

type AutomationCatalogPackageOption struct {
	LineID            int64  `json:"line_id"`
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	CPU               int    `json:"cpu"`
	MemoryGB          int    `json:"memory_gb"`
	DiskGB            int    `json:"disk_gb"`
	BandwidthMbps     int    `json:"bandwidth_mbps"`
	PortNum           int    `json:"port_num"`
	MonthlyPrice      int64  `json:"monthly_price"`
	CapacityRemaining int    `json:"capacity_remaining"`
}

type AutomationCatalogProductOption struct {
	LineID int64  `json:"line_id"`
	ID     int64  `json:"id"`
	Name   string `json:"name"`
}

type AutomationCatalogStringOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type AutomationCatalogOptions struct {
	LineItems         []AutomationCatalogLineOption    `json:"line_items"`
	ProductTypeItems  []AutomationCatalogLineOption    `json:"product_type_items"`
	PackageItems      []AutomationCatalogPackageOption `json:"package_items"`
	ProductItems      []AutomationCatalogProductOption `json:"product_items"`
	BillingCycleItems []AutomationCatalogStringOption  `json:"billing_cycle_items"`
	CancelTypeItems   []AutomationCatalogStringOption  `json:"cancel_type_items"`
}

func (s *Service) ListAutomationCatalogOptions(ctx context.Context, goodsTypeID int64) (AutomationCatalogOptions, error) {
	if s.automation == nil || goodsTypeID <= 0 {
		return AutomationCatalogOptions{}, appshared.ErrInvalidInput
	}
	cli, err := s.automation.ClientForGoodsType(ctx, goodsTypeID)
	if err != nil {
		return AutomationCatalogOptions{}, err
	}
	lines, err := cli.ListLines(ctx)
	if err != nil {
		return AutomationCatalogOptions{}, err
	}

	sort.SliceStable(lines, func(i, j int) bool {
		if lines[i].ID == lines[j].ID {
			return lines[i].Name < lines[j].Name
		}
		return lines[i].ID < lines[j].ID
	})

	lineItems := make([]AutomationCatalogLineOption, 0, len(lines))
	packageItems := make([]AutomationCatalogPackageOption, 0, len(lines)*4)
	productMap := map[int64]AutomationCatalogProductOption{}
	for _, line := range lines {
		lineItems = append(lineItems, AutomationCatalogLineOption{
			ID:     line.ID,
			Name:   line.Name,
			AreaID: line.AreaID,
			State:  line.State,
		})

		products, listErr := cli.ListProducts(ctx, line.ID)
		if listErr != nil {
			return AutomationCatalogOptions{}, listErr
		}
		for _, product := range products {
			if product.ID > 0 {
				if _, exists := productMap[product.ID]; !exists {
					productMap[product.ID] = AutomationCatalogProductOption{
						LineID: line.ID,
						ID:     product.ID,
						Name:   product.Name,
					}
				}
			}
			packageItems = append(packageItems, AutomationCatalogPackageOption{
				LineID:            line.ID,
				ID:                product.ID,
				Name:              product.Name,
				CPU:               product.CPU,
				MemoryGB:          product.MemoryGB,
				DiskGB:            product.DiskGB,
				BandwidthMbps:     product.Bandwidth,
				PortNum:           product.PortNum,
				MonthlyPrice:      product.Price,
				CapacityRemaining: product.CapacityRemaining,
			})
		}
	}

	sort.SliceStable(packageItems, func(i, j int) bool {
		if packageItems[i].LineID == packageItems[j].LineID {
			if packageItems[i].ID == packageItems[j].ID {
				return packageItems[i].Name < packageItems[j].Name
			}
			return packageItems[i].ID < packageItems[j].ID
		}
		return packageItems[i].LineID < packageItems[j].LineID
	})

	productItems := make([]AutomationCatalogProductOption, 0, len(productMap))
	for _, item := range productMap {
		productItems = append(productItems, item)
	}
	sort.SliceStable(productItems, func(i, j int) bool {
		if productItems[i].ID == productItems[j].ID {
			return productItems[i].Name < productItems[j].Name
		}
		return productItems[i].ID < productItems[j].ID
	})

	billingCycleItems := []AutomationCatalogStringOption{
		{Value: "hour", Label: "hour"},
		{Value: "day", Label: "day"},
		{Value: "monthly", Label: "monthly"},
		{Value: "quarterly", Label: "quarterly"},
		{Value: "semiannually", Label: "semiannually"},
		{Value: "annually", Label: "annually"},
		{Value: "biennially", Label: "biennially"},
		{Value: "triennially", Label: "triennially"},
		{Value: "onetime", Label: "onetime"},
		{Value: "free", Label: "free"},
		{Value: "ontrial", Label: "ontrial"},
	}
	cancelTypeItems := []AutomationCatalogStringOption{
		{Value: "Immediate", Label: "Immediate"},
		{Value: "Endofbilling", Label: "Endofbilling"},
	}

	return AutomationCatalogOptions{
		LineItems:         lineItems,
		ProductTypeItems:  lineItems,
		PackageItems:      packageItems,
		ProductItems:      productItems,
		BillingCycleItems: billingCycleItems,
		CancelTypeItems:   cancelTypeItems,
	}, nil
}
