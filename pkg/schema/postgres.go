package schema

import (
	"os"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v2"
)

func (s *Db) LoadPostgres(schemaPath string) error {
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}

	s.Tables, err = parsePostgresSchema(string(schemaBytes))
	if err != nil {
		return err
	}

	return nil
}

func parsePostgresSchema(schemaInput string) (map[string]Table, error) {
	tables := map[string]Table{}
	tree, err := pg_query.Parse(schemaInput)
	if err != nil {
		return nil, err
	}

	for _, stmt := range tree.Stmts {
		if stmt.Stmt == nil {
			continue
		}

		createStmt := stmt.Stmt.GetCreateStmt()
		if createStmt != nil {
			tableName := createStmt.Relation.Relname
			table := Table{
				Name:    tableName,
				Columns: map[string]Column{},
			}

			for _, colElem := range createStmt.TableElts {
				if colElem.GetColumnDef() == nil {
					continue
				}
				colDef := colElem.GetColumnDef()

				typeParts := []string{}
				for _, typNode := range colDef.TypeName.Names {
					if typNode.GetString_() == nil {
						continue
					}
					tStr := typNode.GetString_()
					typeParts = append(typeParts, tStr.Str)
				}

				colName := colDef.Colname
				table.Columns[colName] = Column{
					Name: colName,
					Type: strings.Join(typeParts, "."),
				}
			}

			tables[tableName] = table
		}

		viewStmt := stmt.Stmt.GetViewStmt()
		if viewStmt != nil {
			tableName := viewStmt.View.Relname
			table := Table{
				Name:     tableName,
				Columns:  map[string]Column{},
				ReadOnly: true,
			}

			query := viewStmt.GetQuery()
			if query == nil {
				continue
			}

			selStmt := query.GetSelectStmt()
			if selStmt == nil {
				continue
			}

			for _, item := range selStmt.TargetList {
				resTarget := item.GetResTarget()
				if resTarget == nil {
					continue
				}

				if resTarget.Name != "" {
					table.Columns[resTarget.Name] = Column{
						Name: resTarget.Name,
					}
					continue
				}

				if resTarget.Val == nil {
					continue
				}

				colRef := resTarget.Val.GetColumnRef()
				if colRef == nil {
					// parse only column references when no alias is provided
					continue
				}

				var colField *pg_query.Node
				if len(colRef.Fields) > 0 {
					colField = colRef.Fields[len(colRef.Fields)-1]
				}

				if colField == nil {
					continue
				}

				if colField.GetAStar() != nil {
					// SELECT * - force parsing explicit columns for simplicity
					continue
				}

				if colField.GetString_() == nil {
					continue
				}

				colName := colField.GetString_().GetStr()
				table.Columns[colName] = Column{
					Name: colName,
					Type: "", // type not set, never used for validation
				}
			}

			tables[tableName] = table
		}
	}

	return tables, nil
}
