package utils

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	types "github.com/ashr-tech/csv-migration-tools/types"
)

func ParseAIResponse(resp string) ([]types.ColumnSchema, error) {
	resp = strings.TrimSpace(resp)

	// Remove markdown code blocks
	resp = strings.TrimPrefix(resp, "```json")
	resp = strings.TrimPrefix(resp, "```")
	resp = strings.TrimSuffix(resp, "```")
	resp = strings.TrimSpace(resp)

	// Remove thinking tags (deepseek-r1 specific)
	// Find content between </think> and end, or just remove all <think>...</think> blocks
	if strings.Contains(resp, "<think>") {
		// Find the last </think> tag
		lastThinkEnd := strings.LastIndex(resp, "</think>")
		if lastThinkEnd != -1 {
			// Take everything after </think>
			resp = resp[lastThinkEnd+8:] // 8 is len("</think>")
			resp = strings.TrimSpace(resp)
		}
	}

	cleaned := strings.TrimSpace(resp)

	var schema []types.ColumnSchema
	if err := json.Unmarshal([]byte(cleaned), &schema); err != nil {
		return nil, fmt.Errorf("JSON unmarshal error: %v\nCleaned response: %s", err, cleaned)
	}

	return schema, nil
}

func ReadCSVString(csvString string) ([][]string, error) {
	if len(csvString) == 0 {
		return nil, fmt.Errorf("no CSV data to convert")
	}

	reader := csv.NewReader(strings.NewReader(csvString))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %v", err)
	}

	if len(records) < 1 {
		return nil, fmt.Errorf("CSV has no data")
	}

	return records, nil
}

func ReadCSVFile(path string) (*string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = true       // Allow lazy quotes
	reader.TrimLeadingSpace = true // Trim spaces after delimiters

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must have at least header and one data row")
	}

	headers := records[0]
	dataRows := records[1:]

	var csvBuffer bytes.Buffer
	csvWriter := csv.NewWriter(&csvBuffer)

	csvWriter.Write(headers)

	for i := range len(dataRows) {
		csvWriter.Write(dataRows[i])
	}

	csvWriter.Flush()

	csv := csvBuffer.String()

	return &csv, nil
}

func WriteCSV(path string, records [][]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	return writer.WriteAll(records)
}

func SaveJSON(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func LoadSchemaJSON(path string) ([]types.ColumnSchema, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var schema []types.ColumnSchema
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&schema); err != nil {
		return nil, err
	}

	return schema, nil
}
