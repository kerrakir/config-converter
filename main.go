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

	switch *to {
	case "json":
		cfg = &model.Config{}
		data, err := os.ReadFile(*input)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	case "huawei":
		result := generator.GenerateHuawei(cfg)
		os.WriteFile(*output, []byte(result), 0644)
	case "cisco":
		result := generator.GenerateCisco(cfg)
		os.WriteFile(*output, []byte(result), 0644)
	default:
		fmt.Println("Unsupported output format")
		os.Exit(1)
	}

	fmt.Println("Conversion complete:", *output)
}
