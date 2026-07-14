package printer

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
)

func Print(w io.Writer, data any, format string) error {
	switch format {
	case "json":
		out, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("error formatting JSON: %w", err)
		}
		_, err = w.Write(append(out, '\n'))
		return err

	case "csv":
		return printCSV(w, data)

	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func printCSV(w io.Writer, data any) error {
	val := reflect.Indirect(reflect.ValueOf(data))
	if !val.IsValid() {
		fmt.Fprintln(w, "No data found.")
		return nil
	}

	items, err := extractItems(val)
	if err != nil {
		return fmt.Errorf("error extracting items: %w", err)
	}
	if len(items) == 0 {
		fmt.Fprintln(w, "No data found.")
		return nil
	}

	headers := extractHeaders(items[0])
	rows := extractRows(items)

	cw := csv.NewWriter(w)
	if err := cw.Write(headers); err != nil {
		return fmt.Errorf("error writing headers: %w", err)
	}
	if err := cw.WriteAll(rows); err != nil {
		return fmt.Errorf("error writing rows: %w", err)
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("error flushing csv: %w", err)
	}

	return nil
}

func extractItems(val reflect.Value) ([]reflect.Value, error) {
	if val.Kind() == reflect.Struct {
		return []reflect.Value{val}, nil
	}
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return nil, fmt.Errorf("unsupported data type for printing: %v", val.Kind())
	}

	var items []reflect.Value
	for i := 0; i < val.Len(); i++ {
		items = append(items, reflect.Indirect(val.Index(i)))
	}
	return items, nil
}

func getFieldName(structField reflect.StructField) string {
	jsonTag := structField.Tag.Get("json")
	if jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] == "-" {
			return "-"
		}
		if parts[0] != "" {
			return parts[0]
		}
	}
	return structField.Name
}

func extractHeaders(item reflect.Value) []string {
	typ := item.Type()
	var headers []string

	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		if !structField.IsExported() {
			continue
		}

		name := getFieldName(structField)
		if name == "-" {
			continue
		}

		headers = append(headers, name)
	}
	return headers
}

func extractRows(items []reflect.Value) [][]string {
	var rows [][]string
	for _, item := range items {
		if row := extractRow(item); row != nil {
			rows = append(rows, row)
		}
	}
	return rows
}

func extractRow(item reflect.Value) []string {
	if !item.IsValid() || (item.Kind() == reflect.Ptr && item.IsNil()) {
		return nil
	}

	var row []string
	typ := item.Type()

	for i := 0; i < item.NumField(); i++ {
		structField := typ.Field(i)
		if !structField.IsExported() {
			continue
		}

		name := getFieldName(structField)
		if name == "-" {
			continue
		}

		field := item.Field(i)
		field = reflect.Indirect(field)

		if !field.IsValid() {
			row = append(row, "")
			continue
		}

		row = append(row, fmt.Sprintf("%v", field.Interface()))
	}
	return row
}
