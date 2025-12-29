package types

type ColumnSchema struct {
	Column        string            `json:"column"`
	TargetColumn  string            `json:"target_column,omitempty"`
	Values        []string          `json:"values"`
	ValuesMapping map[string]string `json:"values_mapping,omitempty"`
}
