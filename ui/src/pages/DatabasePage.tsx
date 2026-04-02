import { useState, useRef, useCallback } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import {
  Database,
  Table2,
  Play,
  ChevronLeft,
  ChevronRight,
  AlertCircle,
  Loader2,
  Copy,
  Check,
  Info,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useCompany } from "@/context/CompanyContext";
import { databaseApi, DbColumnInfo } from "@/api/database";
import { Button } from "@/components/ui/button";

// ─── Helpers ─────────────────────────────────────────────────────────────────

function cellValue(v: unknown): string {
  if (v === null || v === undefined) return "NULL";
  if (typeof v === "object") return JSON.stringify(v);
  return String(v);
}

function typeColor(type: string): string {
  const t = type.toLowerCase();
  if (t.includes("int") || t.includes("numeric") || t.includes("real") || t.includes("float") || t.includes("double") || t.includes("decimal")) return "text-blue-400";
  if (t.includes("char") || t.includes("text") || t.includes("clob") || t.includes("varchar")) return "text-green-400";
  if (t.includes("bool")) return "text-purple-400";
  if (t.includes("date") || t.includes("time")) return "text-amber-400";
  if (t.includes("blob") || t.includes("binary")) return "text-pink-400";
  return "text-muted-foreground";
}

// ─── Schema Popover ───────────────────────────────────────────────────────────

function SchemaPopover({ columns }: { columns: DbColumnInfo[] }) {
  return (
    <div className="absolute z-50 top-8 right-0 bg-popover border border-border rounded-lg shadow-lg p-3 min-w-64 text-xs space-y-1">
      <div className="font-medium text-foreground mb-2">Schema</div>
      {columns.map((col) => (
        <div key={col.name} className="flex items-center gap-2">
          {col.key === "PRI" && <span className="text-amber-400 shrink-0">●</span>}
          {col.key !== "PRI" && <span className="text-transparent shrink-0">●</span>}
          <span className="font-mono text-foreground truncate flex-1">{col.name}</span>
          <span className={cn("font-mono shrink-0", typeColor(col.type))}>{col.type}</span>
          {col.nullable && <span className="text-muted-foreground shrink-0">?</span>}
        </div>
      ))}
    </div>
  );
}

// ─── Results Table ────────────────────────────────────────────────────────────

function ResultsTable({ rows, columns }: { rows: Record<string, unknown>[]; columns?: DbColumnInfo[] }) {
  if (rows.length === 0) {
    return (
      <div className="flex items-center justify-center h-24 text-sm text-muted-foreground">
        No rows returned.
      </div>
    );
  }
  const keys = Object.keys(rows[0]);
  return (
    <div className="overflow-auto max-h-[calc(100vh-320px)]">
      <table className="w-full text-xs border-collapse">
        <thead>
          <tr className="sticky top-0 bg-muted/80 backdrop-blur-sm">
            {keys.map((k) => {
              const col = columns?.find((c) => c.name === k);
              return (
                <th
                  key={k}
                  className="text-left px-3 py-2 font-medium text-muted-foreground border-b border-border whitespace-nowrap"
                >
                  <span className="font-mono">{k}</span>
                  {col && (
                    <span className={cn("ml-1.5 font-normal", typeColor(col.type))}>
                      {col.type}
                    </span>
                  )}
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr key={i} className="border-b border-border/50 hover:bg-muted/30 transition-colors">
              {keys.map((k) => {
                const raw = row[k];
                const display = cellValue(raw);
                const isNull = raw === null || raw === undefined;
                return (
                  <td
                    key={k}
                    className={cn(
                      "px-3 py-1.5 font-mono whitespace-nowrap max-w-xs truncate",
                      isNull ? "text-muted-foreground/50 italic" : "text-foreground"
                    )}
                    title={display}
                  >
                    {display}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ─── Table Browser Tab ────────────────────────────────────────────────────────

function TableBrowser({ companyId }: { companyId: string }) {
  const [selectedTable, setSelectedTable] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [showSchema, setShowSchema] = useState(false);

  const { data: tablesData, isLoading: tablesLoading } = useQuery({
    queryKey: ["db", "tables", companyId],
    queryFn: () => databaseApi.listTables(companyId),
  });

  const tables = tablesData?.tables ?? [];

  const { data: schemaData } = useQuery({
    queryKey: ["db", "schema", companyId, selectedTable],
    queryFn: () => databaseApi.tableSchema(companyId, selectedTable!),
    enabled: !!selectedTable,
  });

  const { data: rowsData, isLoading: rowsLoading, isFetching } = useQuery({
    queryKey: ["db", "rows", companyId, selectedTable, page],
    queryFn: () => databaseApi.tableRows(companyId, selectedTable!, page),
    enabled: !!selectedTable,
  });

  function selectTable(t: string) {
    setSelectedTable(t);
    setPage(1);
    setShowSchema(false);
  }

  return (
    <div className="flex h-full min-h-0 gap-0">
      {/* Left — table list */}
      <div className="w-52 shrink-0 border-r border-border flex flex-col min-h-0">
        <div className="px-3 py-2 text-xs font-medium text-muted-foreground border-b border-border">
          Tables
          {tables.length > 0 && (
            <span className="ml-1.5 text-muted-foreground/60">({tables.length})</span>
          )}
        </div>
        <div className="flex-1 overflow-y-auto">
          {tablesLoading ? (
            <div className="flex items-center justify-center h-16">
              <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
            </div>
          ) : tables.length === 0 ? (
            <div className="px-3 py-4 text-xs text-muted-foreground">No tables found.</div>
          ) : (
            tables.map((t) => (
              <button
                key={t}
                onClick={() => selectTable(t)}
                className={cn(
                  "w-full text-left px-3 py-2 text-xs font-mono truncate transition-colors hover:bg-accent/50",
                  selectedTable === t ? "bg-accent text-foreground" : "text-muted-foreground"
                )}
              >
                <Table2 className="h-3 w-3 inline mr-1.5 shrink-0 opacity-60" />
                {t}
              </button>
            ))
          )}
        </div>
      </div>

      {/* Right — rows */}
      <div className="flex-1 min-w-0 flex flex-col min-h-0">
        {!selectedTable ? (
          <div className="flex flex-col items-center justify-center h-full gap-2 text-muted-foreground">
            <Database className="h-8 w-8 opacity-30" />
            <p className="text-sm">Select a table to browse rows</p>
          </div>
        ) : (
          <>
            {/* Table header bar */}
            <div className="flex items-center gap-2 px-4 py-2 border-b border-border shrink-0">
              <span className="font-mono text-sm font-medium">{selectedTable}</span>
              {rowsData && (
                <span className="text-xs text-muted-foreground">
                  {rowsData.total.toLocaleString()} rows
                </span>
              )}
              {(rowsLoading || isFetching) && (
                <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />
              )}
              <div className="ml-auto flex items-center gap-1 relative">
                {schemaData && (
                  <button
                    className="p-1 rounded hover:bg-accent text-muted-foreground"
                    onClick={() => setShowSchema((s) => !s)}
                  >
                    <Info className="h-3.5 w-3.5" />
                  </button>
                )}
                {showSchema && schemaData && (
                  <SchemaPopover columns={schemaData.columns} />
                )}
              </div>
            </div>

            {/* Rows */}
            <div className="flex-1 min-h-0 overflow-hidden">
              {rowsLoading ? (
                <div className="flex items-center justify-center h-24">
                  <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                </div>
              ) : (
                <ResultsTable rows={rowsData?.rows ?? []} columns={schemaData?.columns} />
              )}
            </div>

            {/* Pagination */}
            {rowsData && rowsData.pages > 1 && (
              <div className="flex items-center justify-between px-4 py-2 border-t border-border shrink-0 text-xs text-muted-foreground">
                <span>
                  Page {rowsData.page} of {rowsData.pages}
                </span>
                <div className="flex items-center gap-1">
                  <button
                    disabled={page <= 1}
                    onClick={() => setPage((p) => p - 1)}
                    className="p-1 rounded hover:bg-accent disabled:opacity-40 disabled:cursor-not-allowed"
                  >
                    <ChevronLeft className="h-3.5 w-3.5" />
                  </button>
                  <button
                    disabled={page >= rowsData.pages}
                    onClick={() => setPage((p) => p + 1)}
                    className="p-1 rounded hover:bg-accent disabled:opacity-40 disabled:cursor-not-allowed"
                  >
                    <ChevronRight className="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

// ─── Query Runner Tab ─────────────────────────────────────────────────────────

const EXAMPLE_QUERIES = [
  "SELECT * FROM agents LIMIT 20",
  "SELECT * FROM issues WHERE status != 'done' ORDER BY created_at DESC LIMIT 50",
  "SELECT status, COUNT(*) as count FROM issues GROUP BY status",
  "SELECT name, status, adapter_type FROM agents ORDER BY name",
];

function QueryRunner({ companyId }: { companyId: string }) {
  const [sql, setSql] = useState("");
  const [copied, setCopied] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const query = useMutation({
    mutationFn: (s: string) => databaseApi.runQuery(companyId, s),
  });

  function runQuery() {
    const trimmed = sql.trim();
    if (!trimmed) return;
    query.mutate(trimmed);
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
      e.preventDefault();
      runQuery();
    }
  }

  const copyResults = useCallback(() => {
    if (!query.data) return;
    const rows = query.data.rows;
    if (rows.length === 0) return;
    const keys = Object.keys(rows[0]);
    const csv = [keys.join("\t"), ...rows.map((r) => keys.map((k) => cellValue(r[k])).join("\t"))].join("\n");
    navigator.clipboard.writeText(csv).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }, [query.data]);

  return (
    <div className="flex flex-col h-full min-h-0 gap-0">
      {/* Editor */}
      <div className="shrink-0 border-b border-border">
        <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border/50 bg-muted/30">
          <span className="text-xs text-muted-foreground font-medium">SQL Editor</span>
          <span className="text-xs text-muted-foreground/50">— SELECT only  ·  Ctrl+Enter to run</span>
          <div className="ml-auto flex items-center gap-1">
            {EXAMPLE_QUERIES.map((q, i) => (
              <button
                key={i}
                onClick={() => setSql(q)}
                className="text-xs px-1.5 py-0.5 rounded bg-muted hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
              >
                Example {i + 1}
              </button>
            ))}
          </div>
        </div>
        <div className="relative">
          <textarea
            ref={textareaRef}
            value={sql}
            onChange={(e) => setSql(e.target.value)}
            onKeyDown={handleKeyDown}
            spellCheck={false}
            rows={6}
            placeholder={"SELECT * FROM agents LIMIT 20"}
            className="w-full px-4 py-3 font-mono text-sm bg-transparent text-foreground placeholder:text-muted-foreground/40 resize-none focus:outline-none"
          />
          <div className="absolute bottom-2 right-3 flex items-center gap-2">
            <Button
              size="sm"
              onClick={runQuery}
              disabled={!sql.trim() || query.isPending}
              className="gap-1.5"
            >
              {query.isPending ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Play className="h-3.5 w-3.5" />
              )}
              Run
            </Button>
          </div>
        </div>
      </div>

      {/* Results */}
      <div className="flex-1 min-h-0 flex flex-col overflow-hidden">
        {query.isError ? (
          <div className="flex items-start gap-2 m-4 p-3 rounded-lg border border-destructive/30 bg-destructive/10 text-destructive text-sm">
            <AlertCircle className="h-4 w-4 mt-0.5 shrink-0" />
            <pre className="font-mono text-xs whitespace-pre-wrap">
              {query.error instanceof Error ? query.error.message : "Query failed"}
            </pre>
          </div>
        ) : query.data ? (
          <>
            <div className="flex items-center gap-3 px-4 py-2 border-b border-border shrink-0">
              <span className="text-xs text-muted-foreground">
                {query.data.rowCount} row{query.data.rowCount !== 1 ? "s" : ""}
              </span>
              <span className="text-xs text-muted-foreground">
                {query.data.elapsedMs}ms
              </span>
              {query.data.capped && (
                <span className="text-xs text-amber-500 flex items-center gap-1">
                  <AlertCircle className="h-3 w-3" />
                  Capped at 500 rows
                </span>
              )}
              <button
                onClick={copyResults}
                className="ml-auto flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
                {copied ? "Copied" : "Copy TSV"}
              </button>
            </div>
            <div className="flex-1 overflow-auto">
              <ResultsTable rows={query.data.rows} />
            </div>
          </>
        ) : (
          <div className="flex flex-col items-center justify-center h-full gap-2 text-muted-foreground">
            <Play className="h-8 w-8 opacity-20" />
            <p className="text-sm">Run a query to see results</p>
            <p className="text-xs opacity-60">Only SELECT, WITH, and EXPLAIN are allowed</p>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

type Tab = "tables" | "query";

export function DatabasePage() {
  const { selectedCompanyId } = useCompany();
  const [tab, setTab] = useState<Tab>("tables");

  if (!selectedCompanyId) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
        No company selected.
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Page header */}
      <div className="flex items-center gap-3 px-6 h-14 border-b border-border shrink-0">
        <Database className="h-4 w-4 text-muted-foreground" />
        <h1 className="text-sm font-semibold">Database</h1>
        <div className="flex items-center gap-1 ml-4">
          {(["tables", "query"] as Tab[]).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={cn(
                "px-3 py-1 text-xs rounded-md font-medium transition-colors capitalize",
                tab === t
                  ? "bg-accent text-foreground"
                  : "text-muted-foreground hover:text-foreground hover:bg-accent/50"
              )}
            >
              {t === "tables" ? "Table Browser" : "Query Runner"}
            </button>
          ))}
        </div>
        <div className="ml-auto flex items-center gap-1.5">
          <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium bg-amber-500/10 text-amber-600 border border-amber-500/20">
            read-only
          </span>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 min-h-0 overflow-hidden">
        {tab === "tables" ? (
          <TableBrowser companyId={selectedCompanyId} />
        ) : (
          <QueryRunner companyId={selectedCompanyId} />
        )}
      </div>
    </div>
  );
}
