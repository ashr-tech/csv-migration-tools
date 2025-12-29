package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	types "github.com/ashr-tech/csv-migration-tools/types"
	utils "github.com/ashr-tech/csv-migration-tools/utils"
)

func main() {
	// Usage: go run converter\convert_csv.go

	var sourceDataPath, sourceSchemaPath, targetSchemaPath, schemaName string

	// Ask for input interactively
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Please enter the source data CSV path: ")
	sourceDataPath, _ = reader.ReadString('\n')
	sourceDataPath = strings.TrimSpace(sourceDataPath)

	fmt.Print("Please enter the source schema JSON path: ")
	sourceSchemaPath, _ = reader.ReadString('\n')
	sourceSchemaPath = strings.TrimSpace(sourceSchemaPath)

	fmt.Print("Please enter the target schema JSON path: ")
	targetSchemaPath, _ = reader.ReadString('\n')
	targetSchemaPath = strings.TrimSpace(targetSchemaPath)

	fmt.Print("Please enter a name for the output file: ")
	schemaName, _ = reader.ReadString('\n')
	schemaName = strings.TrimSpace(schemaName)

	// Load schemas
	sourceSchema, err := utils.LoadSchemaJSON(sourceSchemaPath)
	if err != nil {
		log.Fatalf("Error loading source schema: %v", err)
	}

	targetSchema, err := utils.LoadSchemaJSON(targetSchemaPath)
	if err != nil {
		log.Fatalf("Error loading target schema: %v", err)
	}

	// Read CSV data
	csvContent, err := utils.ReadCSVFile(sourceDataPath)
	if err != nil {
		log.Fatalf("Error reading CSV data: %v", err)
	}

	// Convert CSV data
	fmt.Println("Converting CSV data...")
	convertedRecords, err := convertData(*csvContent, sourceSchema, targetSchema)
	if err != nil {
		log.Fatalf("Error converting data: %v", err)
	}

	// Write output CSV
	csvFile := fmt.Sprintf("output/converted_%s.csv", schemaName)
	if err := utils.WriteCSV(csvFile, convertedRecords); err != nil {
		log.Fatalf("Error writing output CSV: %v", err)
	}

	fmt.Printf("✓ Successfully converted %d rows to %s\n", len(convertedRecords)-1, csvFile)
}

func convertData(csvString string, sourceSchema, targetSchema []types.ColumnSchema) ([][]string, error) {
	// Parse CSV string into rows
	records, err := utils.ReadCSVString(csvString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %v", err)
	}

	// Build source column index map
	sourceColIndex := make(map[string]int)
	for i, colName := range records[0] {
		sourceColIndex[strings.TrimSpace(colName)] = i
	}

	// Build output structure
	var output [][]string

	// Create output header from target schema
	outputHeader := make([]string, len(targetSchema))
	for i, col := range targetSchema {
		outputHeader[i] = col.Column
	}
	output = append(output, outputHeader)

	// Convert each data row
	for rowIdx := 1; rowIdx < len(records); rowIdx++ {
		sourceRow := records[rowIdx]
		outputRow := make([]string, len(targetSchema))

		for i, targetCol := range targetSchema {
			value := ""

			// Find corresponding source column in schema
			for _, sourceCol := range sourceSchema {
				if sourceCol.TargetColumn == targetCol.Column {
					// Get value from source row
					if colIdx, exists := sourceColIndex[sourceCol.Column]; exists && colIdx < len(sourceRow) {
						sourceValue := strings.TrimSpace(sourceRow[colIdx])

						if sourceValue != "" {
							// Convert value if mapping exists
							convertedValue := convertValue(sourceValue, sourceCol)
							value = convertedValue
						}
					}
					break
				}
			}

			outputRow[i] = value
		}

		output = append(output, outputRow)
	}

	return output, nil
}

func convertValue(value string, sourceCol types.ColumnSchema) string {
	// If there's a values mapping, apply it
	if sourceCol.ValuesMapping != nil {
		if mappedValue, exists := sourceCol.ValuesMapping[value]; exists {
			return mappedValue
		}
	}

	// Return original value if no mapping found
	return value
}
