package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"converter/generator"
	"converter/model"
	"converter/parser"
)

func main() {
	fmt.Println("Program started")
	input := flag.String("in", "", "Input config file")
	output := flag.String("out", "", "Output config file")
	from := flag.String("from", "cisco", "Input type: cisco|huawei|json")
	to := flag.String("to", "huawei", "Output type: huawei|cisco|json")
	ifMap := flag.String("if-map", "", "Interface type mapping list, e.g. FastEthernet=GigabitEthernet,GigabitEthernet=10GE")
	ifIndex := flag.String("if-index", "keep", "Interface index format: keep|2|3")
	ifIndexPrefix := flag.String("if-index-prefix", "1", "Leading segment for 3-part indexes (e.g. 1 -> 1/0/1)")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Println("Usage: converter -in <file> -out <file> -from cisco -to huawei")
		os.Exit(1)
	}

	var cfg *model.Config
	var err error

	switch *from {
	case "cisco":
		cfg, err = parser.ParseCisco(*input)
	case "huawei":
		cfg, err = parser.ParseHuawei(*input)
	case "json":
		cfg = &model.Config{}
		data, _ := os.ReadFile(*input)
		json.Unmarshal(data, cfg)
	default:
		fmt.Println("Unsupported input format")
		os.Exit(1)
	}

	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	var mappings []interfaceMapping
	if *ifMap != "" {
		mappings, err = parseInterfaceMappings(*ifMap)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	}
	indexStyle := *ifIndex
	if indexStyle != "keep" && indexStyle != "2" && indexStyle != "3" {
		fmt.Println("Error: -if-index must be one of keep|2|3")
		os.Exit(1)
	}
	if len(mappings) > 0 || indexStyle != "keep" {
		applyInterfaceTransformations(cfg, mappings, interfaceTransformOptions{
			indexStyle:      indexStyle,
			threePartPrefix: *ifIndexPrefix,
		})
	}

	switch *to {
	case "json":
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*output, data, 0644); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	case "huawei":
		result := generator.GenerateHuawei(cfg)
		if err := os.WriteFile(*output, []byte(result), 0644); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	case "cisco":
		result := generator.GenerateCisco(cfg)
		if err := os.WriteFile(*output, []byte(result), 0644); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Unsupported output format")
		os.Exit(1)
	}

	fmt.Println("Conversion complete:", *output)
}
