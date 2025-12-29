package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	ai "github.com/ashr-tech/csv-migration-tools/ai"
	types "github.com/ashr-tech/csv-migration-tools/types"
	utils "github.com/ashr-tech/csv-migration-tools/utils"
)

func main() {
	// Usage: go run generator\generate_schemas.go

	// NOTE! Set your Ollama cloud api key first if want to use CLOUD mode
	// $env:OLLAMA_API_KEY="your-api-key-here" (Windows)
	// export OLLAMA_API_KEY="your-api-key-here" (macOS)
	// Get api key: https://ollama.com/settings/keys

	var targetSampleDataPath, sourceSampleDataPath, aiMode, schemaName string

	// Ask for input interactively
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Please enter the source sample CSV path: ")
	sourceSampleDataPath, _ = reader.ReadString('\n')
	sourceSampleDataPath = strings.TrimSpace(sourceSampleDataPath)

	fmt.Print("Please enter the target sample CSV path: ")
	targetSampleDataPath, _ = reader.ReadString('\n')
	targetSampleDataPath = strings.TrimSpace(targetSampleDataPath)

	fmt.Print("Please enter AI mode (CLOUD/LOCAL) [default: CLOUD]: ")
	aiMode, _ = reader.ReadString('\n')
	aiMode = strings.TrimSpace(aiMode)
	if aiMode == "" {
		aiMode = "CLOUD"
	}

	fmt.Print("Please enter a name for the schemas: ")
	schemaName, _ = reader.ReadString('\n')
	schemaName = strings.TrimSpace(schemaName)

	// Generate target schema from target sample data
	fmt.Println("Generating target_schema.json from sample data...")
	targetSchema, err := generateTargetSchema(targetSampleDataPath, &aiMode)
	if err != nil {
		log.Fatalf("Error generating target schema: %v", err)
	}

	// Save target schema
	targetSchemaFile := fmt.Sprintf("output/schemas/target_schema_%s.json", schemaName)
	if err := utils.SaveJSON(targetSchemaFile, targetSchema); err != nil {
		log.Fatalf("Error saving target schema: %v", err)
	}
	fmt.Printf("✓ %s generated successfully", targetSchemaFile)

	// Generate source schema from source sample data and target schema
	fmt.Println("\nGenerating source_schema.json...")
	sourceSchema, err := generateSourceSchema(sourceSampleDataPath, targetSchema, &aiMode)
	if err != nil {
		log.Fatalf("Error generating source schema: %v", err)
	}

	// Save source schema
	sourceSchemaFile := fmt.Sprintf("output/schemas/source_schema_%s.json", schemaName)
	if err := utils.SaveJSON(sourceSchemaFile, sourceSchema); err != nil {
		log.Fatalf("Error saving source schema: %v", err)
	}
	fmt.Printf("✓ %s generated successfully", sourceSchemaFile)
}

func generateTargetSchema(csvPath string, mode *string) ([]types.ColumnSchema, error) {
	csv, err := utils.ReadCSVFile(csvPath)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`
You are a strict data schema (JSON) generator for tabular data analysis.

Analyze ALL columns from the CSV below. The CSV contains complete data - all categorical values that exist are present in the dataset.

CSV DATA:
%s

Return ONLY valid JSON in this format:
[
  {
    "column": "column_name",
    "values": ["value1", "value2"]
  }
]

CLASSIFICATION RULES:
A column is CATEGORICAL (has "values") if values represent:
- Fixed categories, types, or classifications
- Status or state indicators (active/inactive, pending/approved/rejected)
- Boolean flags (true/false, yes/no, Y/N, 1/0)
- Predefined options or enums (role: admin/user/guest, priority: low/medium/high)
- Fixed attributes (size: S/M/L, gender: M/F/Other)

A column is DYNAMIC (empty "values") if values are:
- Unique identifiers (id, uuid, code, reference numbers)
- Names, titles, or descriptive text
- Numeric measurements (price, quantity, amount, score, age)
- Dates and timestamps (created_at, updated_at, birth_date)
- Email addresses, URLs, phone numbers
- Free-text fields (notes, descriptions, comments)
- Foreign key IDs that reference other entities (user_id, product_id, category_id)

CRITICAL RULES FOR RELATIONSHIPS:
- If a column name ends with "_id" (like user_id, store_id, category_id), treat it as DYNAMIC
- If another column exists with the same prefix but different suffix (like user_id + user_name, store_id + store_name), BOTH columns must be DYNAMIC
- Even if these related columns have few unique values, they represent references to other data, not fixed categories

PATTERN DETECTION:
- Columns with paired patterns like (X_id, X_name) or (X_code, X_description) indicate relationships → both DYNAMIC
- Columns ending with _count, _total, _amount, _price, _quantity → always DYNAMIC
- Columns ending with _type, _status, _level, _priority → likely CATEGORICAL

COMPOSITE VALUE HANDLING:
- Composite values may use different separators: "read,write" or "read, write" (with/without spaces)
- Maintain the TARGET SCHEMA separator format in "values_mapping"
- Example: CSV "trx, history" maps to TARGET "transaction,history" (match target format)

OUTPUT REQUIREMENTS:
- Pure JSON only (no markdown, no explanations, no preamble)
- Number of objects MUST equal number of CSV columns
- Preserve exact CSV header names (case-sensitive)
- Maintain CSV column order

EXAMPLE:
[
  {"column": "id", "values": []},
  {"column": "name", "values": []},
  {"column": "email", "values": []},
  {"column": "age", "values": []},
  {"column": "role", "values": ["admin", "manager", "employee"]},
  {"column": "status", "values": ["active", "inactive"]},
  {"column": "department_id", "values": []},
  {"column": "department_name", "values": []},
  {"column": "permissions", "values": ["read", "write", "delete", "read,write", "read,write,delete"]},
  {"column": "created_at", "values": []}
]
`, *csv)

	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Println("GENERATE TARGET SCHEMA PROMPT:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println(prompt)
	fmt.Println(strings.Repeat("-", 80))

	resp, err := ai.CallAI(prompt, mode)
	if err != nil {
		return nil, fmt.Errorf("AI call failed: %v", err)
	}

	fmt.Println("\nGENERATE TARGET SCHEMA AI RESPONSE:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println(resp)
	fmt.Println(strings.Repeat("-", 80))

	// Parse AI response
	schema, err := utils.ParseAIResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %v", err)
	}

	return schema, nil
}

func generateSourceSchema(
	csvPath string,
	targetSchema []types.ColumnSchema,
	mode *string,
) ([]types.ColumnSchema, error) {
	rawCSV, err := utils.ReadCSVFile(csvPath)
	if err != nil {
		return nil, err
	}

	targetSchemaJson, _ := json.MarshalIndent(targetSchema, "", "  ")

	prompt := fmt.Sprintf(`
You are a strict data mapping schema (JSON) generator for tabular data analysis.

Analyze ALL columns from the CSV below and map them to the target schema. The CSV contains complete data - all categorical values that exist are present in the dataset.

CSV DATA:
%s

TARGET SCHEMA JSON:
%s

Return ONLY valid JSON in this format:
[
  {
    "column": "csv_column_name",
    "target_column": "target_column_name",
    "values": ["value1", "value2"],
    "values_mapping": {
      "value1": "target_value1",
      "value2": "target_value2"
    }
  }
]

COLUMN MAPPING RULES:
1. Match CSV columns to TARGET SCHEMA columns based on:
   - Exact or similar names (username → name, active → is_active)
   - Semantic meaning (location_id → store_id, user_role → role)
   - Data type and purpose (both are IDs, both are status fields, etc.)

2. Map each TARGET SCHEMA object to the most appropriate CSV column
3. If no suitable CSV column exists, still include the target_column with "column": null

VALUE CLASSIFICATION RULES:
A column is CATEGORICAL (has "values") if values represent:
- Fixed categories, types, or classifications
- Status or state indicators (active/inactive, pending/approved/rejected)
- Boolean flags (true/false, yes/no, Y/N, 1/0)
- Predefined options or enums (role: admin/user/guest, priority: low/medium/high)
- Fixed attributes (size: S/M/L, gender: M/F/Other)

A column is DYNAMIC (empty "values") if values are:
- Unique identifiers (id, uuid, code, reference numbers)
- Names, titles, or descriptive text
- Numeric measurements (price, quantity, amount, score, age)
- Dates and timestamps (created_at, updated_at, birth_date)
- Email addresses, URLs, phone numbers
- Free-text fields (notes, descriptions, comments)
- Foreign key IDs that reference other entities (user_id, product_id, category_id)

CRITICAL RULES FOR RELATIONSHIPS:
- If a column name ends with "_id" (like user_id, store_id, category_id), treat it as DYNAMIC
- If another column exists with the same prefix but different suffix (like user_id + user_name, store_id + store_name), BOTH columns must be DYNAMIC
- Even if these related columns have few unique values, they represent references to other data, not fixed categories
- Set "values" to empty array [] and "values_mapping" to null for all DYNAMIC columns

PATTERN DETECTION:
- Columns with paired patterns like (X_id, X_name) or (X_code, X_description) indicate relationships → both DYNAMIC
- Columns ending with _count, _total, _amount, _price, _quantity → always DYNAMIC
- Columns ending with _type, _status, _level, _priority → likely CATEGORICAL

COMPOSITE VALUE HANDLING:
- Composite values may use different separators: "read,write" or "read, write" (with/without spaces)
- Maintain the TARGET SCHEMA separator format in "values_mapping"
- Example: CSV "trx, history" maps to TARGET "transaction,history" (match target format)

VALUE MAPPING RULES:
"values_mapping" is ONLY populated when:
1. Both CSV column "values" AND TARGET SCHEMA "values" are NOT empty (both are categorical)
2. Map each CSV value to the closest semantic meaning in TARGET SCHEMA values
3. Consider abbreviations, synonyms, and common variations (Y→true, staff→employee, trx→transaction)
4. For composite values (comma-separated), map each component then reconstruct (trx,history → transaction,history)

Set "values_mapping" to null when:
- CSV column is DYNAMIC (empty "values"), OR
- TARGET SCHEMA column is DYNAMIC (empty "values"), OR
- Both are DYNAMIC

OUTPUT REQUIREMENTS:
- Pure JSON only (no markdown, no explanations, no preamble)
- Number of objects MUST equal number of TARGET SCHEMA objects
- Maintain TARGET SCHEMA object order
- Use exact TARGET SCHEMA column names for "target_column"

EXAMPLE:

CSV: id, username, age, active, user_role, permissions, location_id, location_name
TARGET SCHEMA: id, name, age, is_active, role, permissions, store_id, store_name

[
  {
    "column": "id",
    "target_column": "id",
    "values": [],
    "values_mapping": null
  },
  {
    "column": "username",
    "target_column": "name",
    "values": [],
    "values_mapping": null
  },
  {
    "column": "age",
    "target_column": "age",
    "values": [],
    "values_mapping": null
  },
  {
    "column": "active",
    "target_column": "is_active",
    "values": ["Y", "N"],
    "values_mapping": {
      "Y": "true",
      "N": "false"
    }
  },
  {
    "column": "user_role",
    "target_column": "role",
    "values": ["admin", "manager", "staff"],
    "values_mapping": {
      "admin": "admin",
      "manager": "manager",
      "staff": "employee"
    }
  },
  {
    "column": "permissions",
    "target_column": "permissions",
    "values": ["trx", "history", "setting", "trx,history", "trx, history, setting"],
    "values_mapping": {
      "trx": "transaction",
      "history": "history",
      "setting": "settings",
      "trx,history": "transaction,history",
      "trx, history, setting": "transaction,history,settings"
    }
  },
  {
    "column": "location_id",
    "target_column": "store_id",
    "values": [],
    "values_mapping": null
  },
  {
    "column": "location_name",
    "target_column": "store_name",
    "values": [],
    "values_mapping": null
  }
]
`, *rawCSV, targetSchemaJson)

	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Println("GENERATE SOURCE SCHEMA PROMPT:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println(prompt)
	fmt.Println(strings.Repeat("-", 80))

	resp, err := ai.CallAI(prompt, mode)
	if err != nil {
		return nil, err
	}

	fmt.Println("\nGENERATE SOURCE SCHEMA AI RESPONSE:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println(resp)
	fmt.Println(strings.Repeat("-", 80))

	// Parse AI response
	schema, err := utils.ParseAIResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %v", err)
	}

	return schema, nil
}
