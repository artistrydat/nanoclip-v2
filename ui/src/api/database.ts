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

export const databaseApi = {
  listTables: (companyId: string) =>
    api.get<{ tables: string[] }>(`/companies/${companyId}/database/tables`),

  tableSchema: (companyId: string, table: string) =>
    api.get<DbTableSchema>(`/companies/${companyId}/database/tables/${encodeURIComponent(table)}/schema`),

  tableRows: (companyId: string, table: string, page = 1) =>
    api.get<DbRowsResult>(`/companies/${companyId}/database/tables/${encodeURIComponent(table)}/rows?page=${page}`),

  runQuery: (companyId: string, sql: string) =>
    api.post<DbQueryResult>(`/companies/${companyId}/database/query`, { sql }),
};
