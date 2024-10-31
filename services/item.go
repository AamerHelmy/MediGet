package services

import (
	"fmt"
	"fuzzyTest/excel"
	"fuzzyTest/medName"
	"fuzzyTest/models"
	"fuzzyTest/price32"
	"sort"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ItemSv struct {
	gdb *gorm.DB
}

func NewItemService(db *gorm.DB) *ItemSv {
	return &ItemSv{gdb: db}
}

func (itmSv *ItemSv) UpdateItemsFromExcel(filePath string) (*BulkReport, error) {
	report := &BulkReport{Name: "Base Items report"}

	// read excel file
	rows, err := excel.ReadFile(filePath)
	if err != nil {
		return report, fmt.Errorf("error while reading file: %v", err)
	}
	report.Total = len(rows)

	excelItems := make([]ExcelItemInfo, 0)
	excelNames := make([]string, 0)
	excelSkus := make([]string, 0)
	skusMap := make(map[string]int)
	nameMap := make(map[string]int)

	// parse rows to items
	for i, row := range rows {
		if len(row) == 0 {
			report.Total--
			continue
		}
		if len(row) < 3 {
			report.Failed = append(report.Failed, ExcelItemInfo{
				Row: i + 2, SKU: getSafeString(row[0], " - "), Name: getSafeString(row[1], "empty"), PricePts: 0, Error: fmt.Errorf("short data in row.").Error(),
			})
			continue
		}
		if row[0] == "" || row[1] == "" || row[2] == "" {
			getSafeString(row[0], " - ")
			report.Failed = append(report.Failed, ExcelItemInfo{
				Row: i + 2, SKU: getSafeString(row[0], " - "), Name: getSafeString(row[1], "empty"), PricePts: 0, Error: fmt.Errorf("empty data in row").Error(),
			})
			continue
		}
		if prevRow, exist := skusMap[row[0]]; exist {
			report.Failed = append(report.Failed, ExcelItemInfo{
				Row: i + 2, SKU: getSafeString(row[0], " - "), Name: getSafeString(row[1], "empty"), PricePts: 0, Error: fmt.Errorf("duplicate SKU found at row [%d]", prevRow+2).Error(),
			})
			continue
		}
		if prevRow, exist := nameMap[row[1]]; exist {
			report.Failed = append(report.Failed, ExcelItemInfo{
				Row: i + 2, SKU: getSafeString(row[0], " - "), Name: getSafeString(row[1], "empty"), PricePts: 0, Error: fmt.Errorf("duplicate Name found at row [%d]", prevRow+2).Error(),
			})
			continue
		}
		pts, err := price32.FromStringPounds(row[2])
		if err != nil {
			report.Failed = append(report.Failed, ExcelItemInfo{
				Row: i + 2, SKU: row[0], Name: row[1], PricePts: 0, Error: fmt.Errorf("wrong price: %v", err).Error(),
			})
			continue
		}
		item := ExcelItemInfo{
			Row:      i + 2,
			SKU:      row[0],
			Name:     medName.RemoveExtraSapces(row[1]),
			CleanName: medName.Clean(row[1]),
			PricePts: pts,
		}

		skusMap[row[0]] = i
		nameMap[row[1]] = i
		excelItems = append(excelItems, item)
		excelNames = append(excelNames, row[1])
		excelSkus = append(excelSkus, row[0])
	}

	// collect all exist items with same excelSku in our DB
	existingSkuMap := make(map[string]models.Item)
	var existingSku []models.Item
	if err := itmSv.gdb.Where("sku IN ?", excelSkus).Find(&existingSku).Error; err == nil {
		for _, item := range existingSku {
			existingSkuMap[item.SKU] = item
		}
	}

	// collect all exist items with same excelNames in our DB
	existingNameMap := make(map[string]models.Item)
	var existingNames []models.Item
	if err := itmSv.gdb.Where("name IN ?", excelNames).Find(&existingNames).Error; err == nil {
		for _, item := range existingNames {
			existingNameMap[item.Name] = item
		}
	}

	// prepare update items and new items
	updates := make([]models.Item, 0)
	new := make([]models.Item, 0)
	for _, excelItem := range excelItems {
		if existingSku, ok := existingSkuMap[excelItem.SKU]; ok {
			if existingSku.Name == excelItem.Name && existingSku.PricePts == excelItem.PricePts {
				report.Unchanged++

			} else if existingName, exist := existingNameMap[excelItem.Name]; exist {
				report.Failed = append(report.Failed, ExcelItemInfo{
					Row: excelItem.Row, SKU: excelItem.SKU, Name: excelItem.Name, PricePts: excelItem.PricePts, Error: fmt.Errorf("this name already exists in db at id: %d | sku: %s", existingName.ID, existingName.SKU).Error(),
				})
			} else if existingSku.Name != excelItem.Name || existingSku.PricePts != excelItem.PricePts {
				modelItem := excelItem.ToModelItem()
				updates = append(updates, modelItem)
				report.Updated++
			}
		} else if existingName, exist := existingNameMap[excelItem.Name]; exist {
			report.Failed = append(report.Failed, ExcelItemInfo{
				Row: excelItem.Row, SKU: excelItem.SKU, Name: excelItem.Name, PricePts: excelItem.PricePts, Error: fmt.Errorf("this name already exists in db at id: %d | sku: %s", existingName.ID, existingName.SKU).Error(),
			})
		} else {
			modelItem := excelItem.ToModelItem()
			new = append(new, modelItem)
			report.NewItems++
		}
	}

	// save items need update
	if len(updates) > 0 {
		updtResult := itmSv.gdb.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "sku"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "price_pts"}),
		}).CreateInBatches(updates, 1000)

		if updtResult.Error != nil {
			for _, item := range updates {
				report.Updated--
				report.Failed = append(report.Failed, ExcelItemInfo{SKU: item.SKU, Name: item.Name, PricePts: item.PricePts, Error: updtResult.Error.Error()})
			}
		}
	}

	// save new items
	if len(new) > 0 {
		err := itmSv.gdb.CreateInBatches(new, 100).Error
		if err != nil {
			for _, item := range new {
				report.NewItems--
				report.Failed = append(report.Failed, ExcelItemInfo{SKU: item.SKU, Name: item.Name, PricePts: item.PricePts, Error: err.Error()})
			}
		}
	}

	sort.Slice(report.Failed, func(i, j int) bool {
		return report.Failed[i].Row < report.Failed[j].Row
	})
	return report, nil
}

func (itmSv *ItemSv) CreateBulk(items []models.Item) error {
	result := itmSv.gdb.CreateInBatches(items, 1000)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (itmSv *ItemSv) GetAll() ([]models.Item, error) {
	var items []models.Item
	result := itmSv.gdb.Find(&items)
	return items, result.Error
}

// get all verndors's items for item by id
func (itmSv *ItemSv) GetAllVendorItemsLinkedForItemID(itemID uint) ([]models.VendorItem, error) {
	var vendorItems []models.VendorItem
	err := itmSv.gdb.Where("base_item_id = ? AND is_linked = true", itemID).
		Preload("Vendor").
		Find(&vendorItems).Error
	return vendorItems, err
}