package app

import (
	"github.com/olekukonko/tablewriter"
	"os"
	"reflect"
	"sandwich/config"
)

func printModifiers(features *config.FeatureConfig) {
	if features == nil {
		return
	}
	var data [][]string
	data = append(data, []string{"Feature", "Type", "Enabled"})

	ele := reflect.TypeOf(features).Elem()
	val := reflect.ValueOf(features).Elem()
	for i := 0; i < ele.NumField(); i++ {
		enabled := "No"
		name := ele.Field(i).Name
		switch ele.Field(i).Type.Kind() {
		case reflect.Bool:
			if val.Field(i).Bool() {
				enabled = "Yes"
			}
		case reflect.Struct:
			// 对各个部分判断
			if val.Field(i).IsZero() {
				enabled = "No"
			} else {
				if val.Field(i).FieldByName("Enabled").Bool() {
					enabled = "Yes"
				}
			}
		default:

		}
		if name == "HTTP3" || name == "WebSocket" || name == "Cache" {
			data = append(data, []string{name, "Feature", enabled})
		} else {
			data = append(data, []string{name, "Mod", enabled})
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header(data[0])
	_ = table.Bulk(data[1:])
	_ = table.Render()
}
