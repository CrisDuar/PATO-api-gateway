package services

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

type ViewService struct {
	db *sql.DB
}

func NewViewService(db *sql.DB) *ViewService {
	return &ViewService{db: db}
}

var allowedViews = map[string]bool{
	"vw_ipm_by_domain":                        true,
	"vw_average_deprivations":                 true,
	"vw_deprivations_by_variable":             true,
	"vw_dashboard03_national_poverty":         true,
	"vw_dashboard03_poverty_by_age":           true,
	"vw_dashboard03_deprivation_contribution": true,
	"vw_dimension_contribution":               true,
	"vw_incidence_by_household_head_sex":      true,
	"vw_incidence_by_person_sex":              true,
}

var allowedColumns = map[string]bool{
	"anio":             true,
	"dominio":          true,
	"variable":         true,
	"area_geografica":  true,
	"pais":             true,
	"grupo_erario":     true,
	"valor_porcentaje": true,
	"privacion":        true,
	"tipo_media_pm":    true,
	"dimension":        true,
	"porcentaje":       true,
	"region":           true,
	"departamento":     true,
	"sexo":             true,
}

func (vs *ViewService) GetViewTotalData(viewName string) ([]map[string]interface{}, error) {
	if !allowedViews[viewName] {
		return nil, fmt.Errorf("vista no permitida: %s", viewName)
	}
	query := fmt.Sprintf(
		`SELECT * FROM "%s" ORDER BY "anio" ASC`,
		viewName,
	)

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

// GetDistinctColumnValues devuelve los valores únicos de una columna dentro de una vista,
// usado para poblar filtros (categorías IPM, ubicación geográfica, etc.).
func (vs *ViewService) GetDistinctColumnValues(viewName string, columnName string) ([]string, error) {
	if !allowedViews[viewName] {
		return nil, fmt.Errorf("vista no permitida: %s", viewName)
	}

	if !allowedColumns[columnName] {
		return nil, fmt.Errorf("columna no permitida: %s", columnName)
	}

	query := fmt.Sprintf(
		`SELECT DISTINCT "%s" FROM "%s" WHERE "%s" IS NOT NULL ORDER BY "%s" ASC`,
		columnName,
		viewName,
		columnName,
		columnName,
	)

	rows, err := vs.db.Query(query)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return nil, err
	}
	defer rows.Close()

	values := make([]string, 0)

	for rows.Next() {
		var raw interface{}

		if err := rows.Scan(&raw); err != nil {
			log.Printf("Error scanning row: %v", err)
			return nil, err
		}

		if raw == nil {
			continue
		}

		values = append(values, fmt.Sprintf("%v", raw))
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error with rows: %v", err)
		return nil, err
	}

	return values, nil
}

func (vs *ViewService) GetViewFilteredData(
	viewName string,
	columnName string,
	columnValue string,
) ([]map[string]interface{}, error) {

	if !allowedViews[viewName] {
		return nil, fmt.Errorf("vista no permitida: %s", viewName)
	}

	if !allowedColumns[columnName] {
		return nil, fmt.Errorf("columna no permitida: %s", columnName)
	}

	query := fmt.Sprintf(
		`SELECT * FROM "%s" WHERE "%s" = $1 ORDER BY "anio" ASC`,
		viewName,
		columnName,
	)

	rows, err := vs.db.Query(query, columnValue)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return nil, err
	}
	defer rows.Close()

	return scanRows(rows)
}

// ColumnFilter representa un par columna/valor usado para filtrar una vista.
type ColumnFilter struct {
	ColumnName  string
	ColumnValue string
}

// GetViewFilteredDataMulti filtra una vista por dos o más columnas a la vez
// (ej. dominio + pais + anio), combinando las condiciones con AND.
func (vs *ViewService) GetViewFilteredDataMulti(
	viewName string,
	filters []ColumnFilter,
) ([]map[string]interface{}, error) {

	if !allowedViews[viewName] {
		return nil, fmt.Errorf("vista no permitida: %s", viewName)
	}

	if len(filters) == 0 {
		return nil, fmt.Errorf("se requiere al menos un filtro")
	}

	conditions := make([]string, 0, len(filters))
	args := make([]interface{}, 0, len(filters))

	for i, f := range filters {
		if !allowedColumns[f.ColumnName] {
			return nil, fmt.Errorf("columna no permitida: %s", f.ColumnName)
		}
		conditions = append(conditions, fmt.Sprintf(`"%s" = $%d`, f.ColumnName, i+1))
		args = append(args, f.ColumnValue)
	}

	query := fmt.Sprintf(
		`SELECT * FROM "%s" WHERE %s ORDER BY "anio" ASC`,
		viewName,
		strings.Join(conditions, " AND "),
	)

	rows, err := vs.db.Query(query, args...)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return nil, err
	}
	defer rows.Close()

	return scanRows(rows)
}

// scanRows vuelca las filas de un *sql.Rows en []map[string]interface{},
// usando los nombres de columna como llaves.
func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
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
