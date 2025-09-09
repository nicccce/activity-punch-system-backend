package tool

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"reflect"
)

func ExportToExcel(f *excelize.File, sheet string, data interface{}) error {
	v := reflect.ValueOf(data)

	if v.Kind() != reflect.Slice {
		return fmt.Errorf("data %v不是切片 !", data)
	}
	if v.Len() == 0 {
		return nil
	}
	elemType := v.Index(0).Type()
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("data %v不是结构体切片 !", data)
	}
	if sheet == "" {
		sheet = "Sheet1"
	}
	_, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}
	cols := []int{}
	headers := []string{}

	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		tag := field.Tag.Get("excel")
		if tag == "-" {
			continue
		}
		if tag == "" {
			tag = field.Name
		}
		cols = append(cols, i)
		headers = append(headers, tag)
		cell, err := excelize.CoordinatesToCellName(len(headers), 1)
		if err != nil {
			return err
		}
		err = f.SetCellValue(sheet, cell, tag)
		if err != nil {
			return err
		}
	}

	for row := 0; row < v.Len(); row++ {
		elem := v.Index(row)
		for colIndex, fieldIndex := range cols {
			value := elem.Field(fieldIndex).Interface()
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, row+2)
			err = f.SetCellValue(sheet, cell, value)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
