package services

import (
	"database/sql"
	"log"
)

type ViewService struct {
	db *sql.DB
}

func NewViewService(db *sql.DB) *ViewService {
	return &ViewService{db: db}
}

func (vs *ViewService) GetViewData(viewName string) ([]map[string]interface{}, error) {
	query := "SELECT * FROM " + viewName + " ORDER BY \"anio\" ASC"
	rows, err := vs.db.Query(query)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		log.Printf("Error getting columns: %v", err)
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		rowData := make([]interface{}, len(columns))
		rowPointers := make([]interface{}, len(columns))
		for i := range rowData {
			rowPointers[i] = &rowData[i]
		}

		if err := rows.Scan(rowPointers...); err != nil {
			log.Printf("Error scanning row: %v", err)
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			rowMap[col] = rowData[i]
		}
		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error with rows: %v", err)
		return nil, err
	}

	return results, nil
}
