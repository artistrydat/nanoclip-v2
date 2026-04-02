import { api } from "./client";

export interface DbColumnInfo {
  name: string;
  type: string;
  nullable: boolean;
  key?: string;
  default?: string | null;
}

export interface DbTableSchema {
  table: string;
  columns: DbColumnInfo[];
}

export interface DbRowsResult {
  table: string;
  rows: Record<string, unknown>[];
  total: number;
  page: number;
  limit: number;
  pages: number;
}

export interface DbQueryResult {
  rows: Record<string, unknown>[];
  rowCount: number;
  elapsedMs: number;
  capped: boolean;
}

export interface CreateColumnDef {
  name: string;
  type: string;
  nullable: boolean;
  pk: boolean;
  default?: string;
}

export const COLUMN_TYPES = [
  "INTEGER",
  "INT",
  "BIGINT",
  "TEXT",
  "VARCHAR(255)",
  "VARCHAR(100)",
  "VARCHAR(50)",
  "CHAR(36)",
  "REAL",
  "FLOAT",
  "DOUBLE",
  "NUMERIC",
  "DECIMAL(10,2)",
  "BOOLEAN",
  "DATE",
  "DATETIME",
  "TIMESTAMP",
  "BLOB",
  "JSON",
] as const;

export const databaseApi = {
  listTables: (companyId: string) =>
    api.get<{ tables: string[] }>(`/companies/${companyId}/database/tables`),

  createTable: (companyId: string, name: string, columns: CreateColumnDef[]) =>
    api.post<{ table: string }>(`/companies/${companyId}/database/tables`, { name, columns }),

  tableSchema: (companyId: string, table: string) =>
    api.get<DbTableSchema>(`/companies/${companyId}/database/tables/${encodeURIComponent(table)}/schema`),

  tableRows: (companyId: string, table: string, page = 1) =>
    api.get<DbRowsResult>(`/companies/${companyId}/database/tables/${encodeURIComponent(table)}/rows?page=${page}`),

  insertRow: (companyId: string, table: string, values: Record<string, unknown>) =>
    api.post<{ ok: boolean }>(`/companies/${companyId}/database/tables/${encodeURIComponent(table)}/rows`, { values }),

  updateRow: (companyId: string, table: string, pk_col: string, pk_val: unknown, values: Record<string, unknown>) =>
    api.put<{ ok: boolean }>(`/companies/${companyId}/database/tables/${encodeURIComponent(table)}/rows`, { pk_col, pk_val, values }),

  deleteRow: (companyId: string, table: string, pk_col: string, pk_val: unknown) =>
    api.delete<{ ok: boolean }>(`/companies/${companyId}/database/tables/${encodeURIComponent(table)}/rows?pk_col=${encodeURIComponent(pk_col)}&pk_val=${encodeURIComponent(String(pk_val))}`),

  runQuery: (companyId: string, sql: string) =>
    api.post<DbQueryResult>(`/companies/${companyId}/database/query`, { sql }),
};
