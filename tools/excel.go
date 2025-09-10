package tools

import (
	"fmt"
	"reflect"

	"github.com/xuri/excelize/v2"
)

func ExportToExcel(f *excelize.File, sheet string, data interface{}) error {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("data %v 不是切片", data)
	}
	if v.Len() == 0 {
		return nil
	}

	elemType := v.Index(0).Type()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("data %v 不是结构体切片", data)
	}

	if sheet == "" {
		sheet = "Sheet1"
	}
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}

	type fieldInfo struct {
		index  []int
		header string
	}

	var fields []fieldInfo

	var collect func(t reflect.Type, parent []int)
	collect = func(t reflect.Type, parent []int) {
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)

			if sf.PkgPath != "" {
				continue
			}

			idx := append(append([]int(nil), parent...), i)

			if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
				collect(sf.Type, idx)
				continue
			}

			tag := sf.Tag.Get("excel")
			if tag == "-" {
				continue
			}
			if tag == "" {
				tag = sf.Name
			}

			fields = append(fields, fieldInfo{index: idx, header: tag})
		}
	}

	collect(elemType, nil)

	// 写表头
	for i, fi := range fields {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, cell, fi.header); err != nil {
			return err
		}
	}

	// 写数据行
	for row := 0; row < v.Len(); row++ {
		elem := v.Index(row)

		if elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}

		for colIndex, fi := range fields {
			fv := elem.FieldByIndex(fi.index)

			var value interface{}
			if fv.Kind() == reflect.Ptr {
				if fv.IsNil() {
					value = ""
				} else {
					value = fv.Elem().Interface()
				}
			} else {
				value = fv.Interface()
			}

			cell, err := excelize.CoordinatesToCellName(colIndex+1, row+2)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return err
			}
		}
	}

	return nil
}
