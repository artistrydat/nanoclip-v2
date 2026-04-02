package handlers

import (
        "fmt"
        "net/http"
        "regexp"
        "strconv"
        "strings"
        "time"

        "github.com/gin-gonic/gin"
        "gorm.io/gorm"
        mw "paperclip-go/middleware"
        "paperclip-go/models"
)

func DatabaseRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("/tables", dbListTables(db))
        rg.POST("/tables", dbCreateTable(db))
        rg.GET("/tables/:table/schema", dbTableSchema(db))
        rg.GET("/tables/:table/rows", dbTableRows(db))
        rg.POST("/tables/:table/rows", dbInsertRow(db))
        rg.PUT("/tables/:table/rows", dbUpdateRow(db))
        rg.DELETE("/tables/:table/rows", dbDeleteRow(db))
        rg.POST("/query", dbRunQuery(db))
}

var safeTableNameRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func dbIsMySQL(db *gorm.DB) bool {
        return db.Dialector.Name() == "mysql"
}

func dbListTables(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var tables []string
                if dbIsMySQL(db) {
                        var rows []struct {
                                TableName string `gorm:"column:TABLE_NAME"`
                        }
                        db.Raw("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() ORDER BY TABLE_NAME").Scan(&rows)
                        for _, r := range rows {
                                tables = append(tables, r.TableName)
                        }
                } else {
                        var rows []struct {
                                Name string `gorm:"column:name"`
                        }
                        db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").Scan(&rows)
                        for _, r := range rows {
                                tables = append(tables, r.Name)
                        }
                }
                if tables == nil {
                        tables = []string{}
                }
                c.JSON(http.StatusOK, gin.H{"tables": tables})
        }
}

type dbColumnInfo struct {
        Name     string  `json:"name"`
        Type     string  `json:"type"`
        Nullable bool    `json:"nullable"`
        Key      string  `json:"key,omitempty"`
        Default  *string `json:"default,omitempty"`
}

func dbTableSchema(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                table := c.Param("table")
                if !safeTableNameRe.MatchString(table) {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
                        return
                }
                var columns []dbColumnInfo
                if dbIsMySQL(db) {
                        var rows []struct {
                                ColumnName    string  `gorm:"column:COLUMN_NAME"`
                                DataType      string  `gorm:"column:DATA_TYPE"`
                                IsNullable    string  `gorm:"column:IS_NULLABLE"`
                                ColumnKey     string  `gorm:"column:COLUMN_KEY"`
                                ColumnDefault *string `gorm:"column:COLUMN_DEFAULT"`
                        }
                        db.Raw(
                                "SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_KEY, COLUMN_DEFAULT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION",
                                table,
                        ).Scan(&rows)
                        for _, r := range rows {
                                columns = append(columns, dbColumnInfo{
                                        Name:     r.ColumnName,
                                        Type:     r.DataType,
                                        Nullable: r.IsNullable == "YES",
                                        Key:      r.ColumnKey,
                                        Default:  r.ColumnDefault,
                                })
                        }
                } else {
                        var rows []struct {
                                Name      string  `gorm:"column:name"`
                                Type      string  `gorm:"column:type"`
                                NotNull   int     `gorm:"column:notnull"`
                                DfltValue *string `gorm:"column:dflt_value"`
                                Pk        int     `gorm:"column:pk"`
                        }
                        db.Raw(fmt.Sprintf("PRAGMA table_info(`%s`)", table)).Scan(&rows)
                        for _, r := range rows {
                                key := ""
                                if r.Pk > 0 {
                                        key = "PRI"
                                }
                                columns = append(columns, dbColumnInfo{
                                        Name:     r.Name,
                                        Type:     r.Type,
                                        Nullable: r.NotNull == 0,
                                        Key:      key,
                                        Default:  r.DfltValue,
                                })
                        }
                }
                if columns == nil {
                        columns = []dbColumnInfo{}
                }
                c.JSON(http.StatusOK, gin.H{"table": table, "columns": columns})
        }
}

func dbTableRows(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                table := c.Param("table")
                if !safeTableNameRe.MatchString(table) {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
                        return
                }
                page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
                if page < 1 {
                        page = 1
                }
                const pageSize = 50
                offset := (page - 1) * pageSize

                var total int64
                db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table)).Scan(&total)

                var rows []map[string]interface{}
                db.Raw(fmt.Sprintf("SELECT * FROM `%s` LIMIT %d OFFSET %d", table, pageSize, offset)).Scan(&rows)
                if rows == nil {
                        rows = []map[string]interface{}{}
                }

                pages := (int(total) + pageSize - 1) / pageSize
                if pages < 1 {
                        pages = 1
                }
                c.JSON(http.StatusOK, gin.H{
                        "table": table,
                        "rows":  rows,
                        "total": total,
                        "page":  page,
                        "limit": pageSize,
                        "pages": pages,
                })
        }
}

// ─── Write helpers ────────────────────────────────────────────────────────────

var safeColNameRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

var safeColTypes = map[string]struct{}{
        "INTEGER": {}, "INT": {}, "BIGINT": {}, "SMALLINT": {}, "TINYINT": {},
        "TEXT": {}, "VARCHAR(50)": {}, "VARCHAR(100)": {}, "VARCHAR(255)": {}, "CHAR(36)": {},
        "REAL": {}, "FLOAT": {}, "DOUBLE": {}, "NUMERIC": {}, "DECIMAL(10,2)": {},
        "BOOLEAN": {}, "BOOL": {},
        "DATE": {}, "DATETIME": {}, "TIMESTAMP": {},
        "BLOB": {}, "JSON": {},
}

type createColDef struct {
        Name     string `json:"name"`
        Type     string `json:"type"`
        Nullable bool   `json:"nullable"`
        PK       bool   `json:"pk"`
        Default  string `json:"default,omitempty"`
}

func dbCreateTable(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                var req struct {
                        Name    string         `json:"name"`
                        Columns []createColDef `json:"columns"`
                }
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
                        return
                }
                if !safeTableNameRe.MatchString(req.Name) {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
                        return
                }
                if len(req.Columns) == 0 {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "at least one column required"})
                        return
                }
                var colDefs []string
                for _, col := range req.Columns {
                        if !safeColNameRe.MatchString(col.Name) {
                                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid column name: " + col.Name})
                                return
                        }
                        if _, ok := safeColTypes[col.Type]; !ok {
                                c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported column type: " + col.Type})
                                return
                        }
                        def := fmt.Sprintf("`%s` %s", col.Name, col.Type)
                        if col.PK {
                                def += " PRIMARY KEY"
                                if col.Type == "INTEGER" || col.Type == "INT" || col.Type == "BIGINT" {
                                        if dbIsMySQL(db) {
                                                def += " AUTO_INCREMENT"
                                        } else {
                                                def += " AUTOINCREMENT"
                                        }
                                }
                        } else if !col.Nullable {
                                def += " NOT NULL"
                        }
                        if col.Default != "" && !col.PK {
                                def += fmt.Sprintf(" DEFAULT '%s'", strings.ReplaceAll(col.Default, "'", "''"))
                        }
                        colDefs = append(colDefs, def)
                }
                sql := fmt.Sprintf("CREATE TABLE `%s` (%s)", req.Name, strings.Join(colDefs, ", "))
                if err := db.Exec(sql).Error; err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                actor := mw.GetActor(c)
                logActivity(db, companyID, actor, "created", "db_table", req.Name, models.JSON{"sql": sql})
                c.JSON(http.StatusCreated, gin.H{"table": req.Name})
        }
}

func dbInsertRow(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                table := c.Param("table")
                if !safeTableNameRe.MatchString(table) {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
                        return
                }
                var req struct {
                        Values map[string]interface{} `json:"values"`
                }
                if err := c.ShouldBindJSON(&req); err != nil || len(req.Values) == 0 {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "missing values"})
                        return
                }
                var cols []string
                var placeholders []string
                var vals []interface{}
                for col, val := range req.Values {
                        if !safeColNameRe.MatchString(col) {
                                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid column name: " + col})
                                return
                        }
                        cols = append(cols, fmt.Sprintf("`%s`", col))
                        placeholders = append(placeholders, "?")
                        vals = append(vals, val)
                }
                sql := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
                        table,
                        strings.Join(cols, ", "),
                        strings.Join(placeholders, ", "),
                )
                if err := db.Exec(sql, vals...).Error; err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                actor := mw.GetActor(c)
                logActivity(db, companyID, actor, "inserted", "db_row", table, models.JSON{"cols": cols})
                c.JSON(http.StatusCreated, gin.H{"ok": true})
        }
}

func dbUpdateRow(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                table := c.Param("table")
                if !safeTableNameRe.MatchString(table) {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
                        return
                }
                var req struct {
                        PKCol  string                 `json:"pk_col"`
                        PKVal  interface{}            `json:"pk_val"`
                        Values map[string]interface{} `json:"values"`
                }
                if err := c.ShouldBindJSON(&req); err != nil || req.PKCol == "" || len(req.Values) == 0 {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "missing pk_col or values"})
                        return
                }
                if !safeColNameRe.MatchString(req.PKCol) {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pk_col"})
                        return
                }
                var setClauses []string
                var vals []interface{}
                for col, val := range req.Values {
                        if !safeColNameRe.MatchString(col) {
                                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid column name: " + col})
                                return
                        }
                        setClauses = append(setClauses, fmt.Sprintf("`%s` = ?", col))
                        vals = append(vals, val)
                }
                vals = append(vals, req.PKVal)
                sql := fmt.Sprintf("UPDATE `%s` SET %s WHERE `%s` = ?",
                        table,
                        strings.Join(setClauses, ", "),
                        req.PKCol,
                )
                if err := db.Exec(sql, vals...).Error; err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                actor := mw.GetActor(c)
                logActivity(db, companyID, actor, "updated", "db_row", table, models.JSON{"pk_col": req.PKCol})
                c.JSON(http.StatusOK, gin.H{"ok": true})
        }
}

func dbDeleteRow(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                table := c.Param("table")
                if !safeTableNameRe.MatchString(table) {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
                        return
                }
                pkCol := c.Query("pk_col")
                pkVal := c.Query("pk_val")
                if pkCol == "" || pkVal == "" {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "missing pk_col or pk_val query params"})
                        return
                }
                if !safeColNameRe.MatchString(pkCol) {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pk_col"})
                        return
                }
                sql := fmt.Sprintf("DELETE FROM `%s` WHERE `%s` = ?", table, pkCol)
                if err := db.Exec(sql, pkVal).Error; err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                actor := mw.GetActor(c)
                logActivity(db, companyID, actor, "deleted", "db_row", table, models.JSON{"pk_col": pkCol, "pk_val": pkVal})
                c.JSON(http.StatusOK, gin.H{"ok": true})
        }
}

var dbForbiddenRe = regexp.MustCompile(`(?i)^\s*(insert|update|delete|drop|alter|create|truncate|replace|grant|revoke|execute|merge|load|lock)`)

func dbRunQuery(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                var req struct {
                        SQL string `json:"sql"`
                }
                if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.SQL) == "" {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "missing sql field"})
                        return
                }
                sql := strings.TrimSpace(req.SQL)

                if dbForbiddenRe.MatchString(sql) {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "only read-only queries (SELECT, WITH, EXPLAIN) are allowed"})
                        return
                }
                fields := strings.Fields(sql)
                if len(fields) == 0 {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "empty query"})
                        return
                }
                first := strings.ToLower(fields[0])
                if first != "select" && first != "with" && first != "explain" {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "query must begin with SELECT, WITH, or EXPLAIN"})
                        return
                }

                // Enforce row cap — append LIMIT if missing
                const hardLimit = 500
                if !strings.Contains(strings.ToLower(sql), " limit ") {
                        sql = sql + fmt.Sprintf(" LIMIT %d", hardLimit)
                }

                start := time.Now()
                var rows []map[string]interface{}
                result := db.Raw(sql).Scan(&rows)
                elapsed := time.Since(start).Milliseconds()

                if result.Error != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
                        return
                }
                if rows == nil {
                        rows = []map[string]interface{}{}
                }

                actor := mw.GetActor(c)
                logActivity(db, companyID, actor, "ran", "db_query", "", models.JSON{
                        "rowCount":  len(rows),
                        "elapsedMs": elapsed,
                })

                c.JSON(http.StatusOK, gin.H{
                        "rows":      rows,
                        "rowCount":  len(rows),
                        "elapsedMs": elapsed,
                        "capped":    len(rows) >= hardLimit,
                })
        }
}
