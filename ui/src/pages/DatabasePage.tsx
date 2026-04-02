import { useState, useRef, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
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
  Plus,
  Pencil,
  Trash2,
  X,
  GripVertical,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useCompany } from "@/context/CompanyContext";
import { databaseApi, DbColumnInfo, CreateColumnDef, COLUMN_TYPES } from "@/api/database";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

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
  if (t.includes("blob") || t.includes("binary") || t.includes("json")) return "text-pink-400";
  return "text-muted-foreground";
}

function isNumericType(type: string) {
  const t = type.toUpperCase();
  return ["INTEGER", "INT", "BIGINT", "SMALLINT", "TINYINT", "REAL", "FLOAT", "DOUBLE", "NUMERIC", "DECIMAL(10,2)"].some((x) => t.startsWith(x.split("(")[0]));
}

function isDateType(type: string) {
  return ["DATE"].includes(type.toUpperCase());
}

function isDateTimeType(type: string) {
  return ["DATETIME", "TIMESTAMP"].includes(type.toUpperCase());
}

function isBoolType(type: string) {
  return ["BOOLEAN", "BOOL"].includes(type.toUpperCase());
}

function getPkCol(columns: DbColumnInfo[]): DbColumnInfo | undefined {
  return columns.find((c) => c.key === "PRI");
}

// ─── Schema Popover ───────────────────────────────────────────────────────────

function SchemaPopover({ columns }: { columns: DbColumnInfo[] }) {
  return (
    <div className="absolute z-50 top-8 right-0 bg-popover border border-border rounded-lg shadow-lg p-3 min-w-64 text-xs space-y-1">
      <div className="font-medium text-foreground mb-2">Schema</div>
      {columns.map((col) => (
        <div key={col.name} className="flex items-center gap-2">
          <span className={col.key === "PRI" ? "text-amber-400 shrink-0" : "text-transparent shrink-0"}>●</span>
          <span className="font-mono text-foreground truncate flex-1">{col.name}</span>
          <span className={cn("font-mono shrink-0", typeColor(col.type))}>{col.type}</span>
          {col.nullable && <span className="text-muted-foreground shrink-0">?</span>}
        </div>
      ))}
    </div>
  );
}

// ─── Row Form Modal ───────────────────────────────────────────────────────────

interface RowFormModalProps {
  mode: "add" | "edit";
  table: string;
  columns: DbColumnInfo[];
  initialValues?: Record<string, unknown>;
  onClose: () => void;
  onSubmit: (values: Record<string, unknown>) => void;
  isPending: boolean;
  error?: string | null;
}

function RowFormModal({ mode, table, columns, initialValues, onClose, onSubmit, isPending, error }: RowFormModalProps) {
  const pkCol = getPkCol(columns);
  const editableCols = mode === "add"
    ? columns.filter((c) => {
        if (c.key === "PRI" && isNumericType(c.type)) return false;
        return true;
      })
    : columns;

  const [values, setValues] = useState<Record<string, unknown>>(() => {
    const init: Record<string, unknown> = {};
    for (const col of editableCols) {
      if (initialValues && col.name in initialValues) {
        init[col.name] = initialValues[col.name] ?? "";
      } else {
        init[col.name] = "";
      }
    }
    return init;
  });

  function setValue(col: string, val: unknown) {
    setValues((prev) => ({ ...prev, [col]: val }));
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const payload: Record<string, unknown> = {};
    for (const col of editableCols) {
      if (mode === "edit" && col.key === "PRI") continue;
      const v = values[col.name];
      if (v === "" || v === null || v === undefined) {
        if (col.nullable) { payload[col.name] = null; continue; }
      }
      if (isBoolType(col.type)) {
        payload[col.name] = Boolean(v);
      } else if (isNumericType(col.type)) {
        payload[col.name] = v === "" ? null : Number(v);
      } else {
        payload[col.name] = v;
      }
    }
    onSubmit(payload);
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-background border border-border rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] flex flex-col">
        <div className="flex items-center justify-between px-5 py-4 border-b border-border shrink-0">
          <div>
            <h2 className="font-semibold text-sm">{mode === "add" ? "Add Row" : "Edit Row"}</h2>
            <p className="text-xs text-muted-foreground font-mono mt-0.5">{table}</p>
          </div>
          <button onClick={onClose} className="p-1 rounded hover:bg-accent text-muted-foreground">
            <X className="h-4 w-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col min-h-0">
          <div className="flex-1 overflow-y-auto px-5 py-4 space-y-4">
            {error && (
              <div className="flex items-start gap-2 p-3 rounded-lg border border-destructive/30 bg-destructive/10 text-destructive text-xs">
                <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
                {error}
              </div>
            )}
            {editableCols.map((col) => {
              const isPK = col.key === "PRI";
              const isReadOnly = mode === "edit" && isPK;
              const val = values[col.name] ?? "";
              return (
                <div key={col.name} className="space-y-1.5">
                  <Label className="flex items-center gap-1.5 text-xs">
                    <span className="font-mono">{col.name}</span>
                    <span className={cn("text-[10px]", typeColor(col.type))}>{col.type}</span>
                    {col.nullable && <span className="text-muted-foreground text-[10px]">nullable</span>}
                    {isPK && <span className="text-amber-500 text-[10px]">PK</span>}
                  </Label>
                  {isBoolType(col.type) ? (
                    <div className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={Boolean(val)}
                        disabled={isReadOnly}
                        onChange={(e) => setValue(col.name, e.target.checked)}
                        className="h-4 w-4 rounded border border-input"
                      />
                      <span className="text-xs text-muted-foreground">{Boolean(val) ? "true" : "false"}</span>
                    </div>
                  ) : (
                    <Input
                      type={isNumericType(col.type) ? "number" : isDateType(col.type) ? "date" : isDateTimeType(col.type) ? "datetime-local" : "text"}
                      value={String(val)}
                      readOnly={isReadOnly}
                      disabled={isReadOnly}
                      placeholder={col.nullable ? "NULL" : ""}
                      className={cn("font-mono text-xs h-8", isReadOnly && "opacity-50 cursor-not-allowed")}
                      onChange={(e) => setValue(col.name, e.target.value)}
                    />
                  )}
                </div>
              );
            })}
          </div>

          <div className="flex items-center justify-end gap-2 px-5 py-4 border-t border-border shrink-0">
            <Button type="button" variant="outline" size="sm" onClick={onClose} disabled={isPending}>
              Cancel
            </Button>
            <Button type="submit" size="sm" disabled={isPending} className="gap-1.5">
              {isPending && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
              {mode === "add" ? "Insert Row" : "Save Changes"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── Create Table Modal ───────────────────────────────────────────────────────

interface ColDraft extends CreateColumnDef {
  id: number;
}

let colIdSeq = 0;
function newCol(): ColDraft {
  return { id: ++colIdSeq, name: "", type: "TEXT", nullable: true, pk: false, default: "" };
}

interface CreateTableModalProps {
  onClose: () => void;
  onSubmit: (name: string, columns: CreateColumnDef[]) => void;
  isPending: boolean;
  error?: string | null;
}

function CreateTableModal({ onClose, onSubmit, isPending, error }: CreateTableModalProps) {
  const [tableName, setTableName] = useState("");
  const [cols, setCols] = useState<ColDraft[]>([{ id: ++colIdSeq, name: "id", type: "INTEGER", nullable: false, pk: true, default: "" }]);

  function addCol() {
    setCols((prev) => [...prev, newCol()]);
  }

  function removeCol(id: number) {
    setCols((prev) => prev.filter((c) => c.id !== id));
  }

  function updateCol(id: number, patch: Partial<ColDraft>) {
    setCols((prev) => prev.map((c) => (c.id === id ? { ...c, ...patch } : c)));
  }

  function setPK(id: number) {
    setCols((prev) => prev.map((c) => ({ ...c, pk: c.id === id })));
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    onSubmit(tableName.trim(), cols.map(({ id: _id, ...rest }) => rest));
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-background border border-border rounded-xl shadow-xl w-full max-w-2xl max-h-[90vh] flex flex-col">
        <div className="flex items-center justify-between px-5 py-4 border-b border-border shrink-0">
          <h2 className="font-semibold text-sm">Create Table</h2>
          <button onClick={onClose} className="p-1 rounded hover:bg-accent text-muted-foreground">
            <X className="h-4 w-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col min-h-0">
          <div className="flex-1 overflow-y-auto px-5 py-4 space-y-5">
            {error && (
              <div className="flex items-start gap-2 p-3 rounded-lg border border-destructive/30 bg-destructive/10 text-destructive text-xs">
                <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
                {error}
              </div>
            )}

            <div className="space-y-1.5">
              <Label className="text-xs">Table Name</Label>
              <Input
                value={tableName}
                onChange={(e) => setTableName(e.target.value)}
                placeholder="my_table"
                className="font-mono text-xs h-8 w-72"
                pattern="[a-zA-Z_][a-zA-Z0-9_]*"
                required
              />
              <p className="text-[10px] text-muted-foreground">Letters, numbers, and underscores only</p>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label className="text-xs">Columns</Label>
                <button type="button" onClick={addCol} className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors">
                  <Plus className="h-3 w-3" /> Add column
                </button>
              </div>

              {/* Header */}
              <div className="grid grid-cols-[16px_1fr_140px_60px_60px_32px] gap-2 px-2 text-[10px] text-muted-foreground font-medium">
                <span></span>
                <span>Name</span>
                <span>Type</span>
                <span className="text-center">Nullable</span>
                <span className="text-center">PK</span>
                <span></span>
              </div>

              <div className="space-y-1.5">
                {cols.map((col) => (
                  <div key={col.id} className="grid grid-cols-[16px_1fr_140px_60px_60px_32px] gap-2 items-center px-2 py-1 rounded-lg hover:bg-muted/30">
                    <GripVertical className="h-3.5 w-3.5 text-muted-foreground/40 cursor-grab" />
                    <Input
                      value={col.name}
                      onChange={(e) => updateCol(col.id, { name: e.target.value })}
                      placeholder="column_name"
                      className="font-mono text-xs h-7"
                      required
                    />
                    <select
                      value={col.type}
                      onChange={(e) => updateCol(col.id, { type: e.target.value })}
                      className="h-7 rounded-md border border-input bg-transparent px-2 text-xs font-mono focus:outline-none focus:ring-1 focus:ring-ring"
                    >
                      {COLUMN_TYPES.map((t) => (
                        <option key={t} value={t}>{t}</option>
                      ))}
                    </select>
                    <div className="flex items-center justify-center">
                      <input
                        type="checkbox"
                        checked={col.nullable}
                        disabled={col.pk}
                        onChange={(e) => updateCol(col.id, { nullable: e.target.checked })}
                        className="h-4 w-4 rounded border border-input"
                      />
                    </div>
                    <div className="flex items-center justify-center">
                      <input
                        type="radio"
                        name="pk_col"
                        checked={col.pk}
                        onChange={() => setPK(col.id)}
                        className="h-4 w-4 border border-input"
                      />
                    </div>
                    <button
                      type="button"
                      onClick={() => removeCol(col.id)}
                      disabled={cols.length <= 1}
                      className="flex items-center justify-center p-1 rounded hover:bg-destructive/10 hover:text-destructive text-muted-foreground disabled:opacity-30 disabled:cursor-not-allowed"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className="flex items-center justify-end gap-2 px-5 py-4 border-t border-border shrink-0">
            <Button type="button" variant="outline" size="sm" onClick={onClose} disabled={isPending}>
              Cancel
            </Button>
            <Button type="submit" size="sm" disabled={isPending || !tableName.trim() || cols.length === 0} className="gap-1.5">
              {isPending && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
              Create Table
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── Confirm Delete Dialog ─────────────────────────────────────────────────────

function ConfirmDeleteDialog({ onConfirm, onCancel, isPending }: { onConfirm: () => void; onCancel: () => void; isPending: boolean }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-background border border-border rounded-xl shadow-xl w-80 p-5 space-y-4">
        <div className="flex items-start gap-3">
          <div className="p-2 rounded-lg bg-destructive/10">
            <Trash2 className="h-4 w-4 text-destructive" />
          </div>
          <div>
            <h3 className="font-semibold text-sm">Delete Row</h3>
            <p className="text-xs text-muted-foreground mt-1">This action cannot be undone. The row will be permanently removed from the database.</p>
          </div>
        </div>
        <div className="flex items-center justify-end gap-2">
          <Button variant="outline" size="sm" onClick={onCancel} disabled={isPending}>Cancel</Button>
          <Button variant="destructive" size="sm" onClick={onConfirm} disabled={isPending} className="gap-1.5">
            {isPending && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
            Delete
          </Button>
        </div>
      </div>
    </div>
  );
}

// ─── Results Table ────────────────────────────────────────────────────────────

interface ResultsTableProps {
  rows: Record<string, unknown>[];
  columns?: DbColumnInfo[];
  onEdit?: (row: Record<string, unknown>) => void;
  onDelete?: (row: Record<string, unknown>) => void;
}

function ResultsTable({ rows, columns, onEdit, onDelete }: ResultsTableProps) {
  if (rows.length === 0) {
    return (
      <div className="flex items-center justify-center h-24 text-sm text-muted-foreground">
        No rows returned.
      </div>
    );
  }
  const keys = Object.keys(rows[0]);
  const showActions = !!(onEdit || onDelete);

  return (
    <div className="overflow-auto max-h-[calc(100vh-320px)]">
      <table className="w-full text-xs border-collapse">
        <thead>
          <tr className="sticky top-0 bg-muted/80 backdrop-blur-sm">
            {keys.map((k) => {
              const col = columns?.find((c) => c.name === k);
              return (
                <th key={k} className="text-left px-3 py-2 font-medium text-muted-foreground border-b border-border whitespace-nowrap">
                  <span className="font-mono">{k}</span>
                  {col && <span className={cn("ml-1.5 font-normal", typeColor(col.type))}>{col.type}</span>}
                </th>
              );
            })}
            {showActions && (
              <th className="w-16 px-2 py-2 border-b border-border sticky right-0 bg-muted/80 backdrop-blur-sm" />
            )}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr key={i} className="border-b border-border/50 hover:bg-muted/30 transition-colors group">
              {keys.map((k) => {
                const raw = row[k];
                const display = cellValue(raw);
                const isNull = raw === null || raw === undefined;
                return (
                  <td
                    key={k}
                    className={cn("px-3 py-1.5 font-mono whitespace-nowrap max-w-xs truncate", isNull ? "text-muted-foreground/50 italic" : "text-foreground")}
                    title={display}
                  >
                    {display}
                  </td>
                );
              })}
              {showActions && (
                <td className="px-2 py-1 sticky right-0 bg-background group-hover:bg-muted/30 transition-colors">
                  <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                    {onEdit && (
                      <button onClick={() => onEdit(row)} className="p-1 rounded hover:bg-accent text-muted-foreground hover:text-foreground transition-colors">
                        <Pencil className="h-3 w-3" />
                      </button>
                    )}
                    {onDelete && (
                      <button onClick={() => onDelete(row)} className="p-1 rounded hover:bg-destructive/10 hover:text-destructive text-muted-foreground transition-colors">
                        <Trash2 className="h-3 w-3" />
                      </button>
                    )}
                  </div>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ─── Table Browser Tab ────────────────────────────────────────────────────────

function TableBrowser({ companyId }: { companyId: string }) {
  const qc = useQueryClient();
  const [selectedTable, setSelectedTable] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [showSchema, setShowSchema] = useState(false);

  // Modals
  const [showCreateTable, setShowCreateTable] = useState(false);
  const [addRowOpen, setAddRowOpen] = useState(false);
  const [editRow, setEditRow] = useState<Record<string, unknown> | null>(null);
  const [deleteRow, setDeleteRow] = useState<Record<string, unknown> | null>(null);

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

  function invalidateRows() {
    qc.invalidateQueries({ queryKey: ["db", "rows", companyId, selectedTable] });
  }

  function invalidateTables() {
    qc.invalidateQueries({ queryKey: ["db", "tables", companyId] });
  }

  // Create table mutation
  const createTable = useMutation({
    mutationFn: ({ name, columns }: { name: string; columns: CreateColumnDef[] }) =>
      databaseApi.createTable(companyId, name, columns),
    onSuccess: (data) => {
      invalidateTables();
      setShowCreateTable(false);
      setSelectedTable(data.table);
    },
  });

  // Insert mutation
  const insertRow = useMutation({
    mutationFn: (values: Record<string, unknown>) =>
      databaseApi.insertRow(companyId, selectedTable!, values),
    onSuccess: () => { invalidateRows(); setAddRowOpen(false); },
  });

  // Update mutation
  const updateRow = useMutation({
    mutationFn: ({ pkCol, pkVal, values }: { pkCol: string; pkVal: unknown; values: Record<string, unknown> }) =>
      databaseApi.updateRow(companyId, selectedTable!, pkCol, pkVal, values),
    onSuccess: () => { invalidateRows(); setEditRow(null); },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: ({ pkCol, pkVal }: { pkCol: string; pkVal: unknown }) =>
      databaseApi.deleteRow(companyId, selectedTable!, pkCol, pkVal),
    onSuccess: () => { invalidateRows(); setDeleteRow(null); },
  });

  function selectTable(t: string) {
    setSelectedTable(t);
    setPage(1);
    setShowSchema(false);
  }

  const pkCol = schemaData ? getPkCol(schemaData.columns) : undefined;

  function handleEditSubmit(values: Record<string, unknown>) {
    if (!editRow || !pkCol) return;
    updateRow.mutate({ pkCol: pkCol.name, pkVal: editRow[pkCol.name], values });
  }

  function handleDeleteConfirm() {
    if (!deleteRow || !pkCol) return;
    deleteMutation.mutate({ pkCol: pkCol.name, pkVal: deleteRow[pkCol.name] });
  }

  return (
    <div className="flex h-full min-h-0 gap-0">
      {/* Left — table list */}
      <div className="w-52 shrink-0 border-r border-border flex flex-col min-h-0">
        <div className="flex items-center px-3 py-2 border-b border-border">
          <span className="text-xs font-medium text-muted-foreground flex-1">
            Tables{tables.length > 0 && <span className="ml-1 opacity-60">({tables.length})</span>}
          </span>
          <button
            onClick={() => setShowCreateTable(true)}
            title="Create table"
            className="p-0.5 rounded hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
          >
            <Plus className="h-3.5 w-3.5" />
          </button>
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
            <div className="flex items-center gap-2 px-4 py-2 border-b border-border shrink-0">
              <span className="font-mono text-sm font-medium">{selectedTable}</span>
              {rowsData && (
                <span className="text-xs text-muted-foreground">{rowsData.total.toLocaleString()} rows</span>
              )}
              {(rowsLoading || isFetching) && (
                <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />
              )}
              <div className="ml-auto flex items-center gap-1 relative">
                {schemaData && pkCol && (
                  <Button size="sm" variant="outline" className="h-7 text-xs gap-1.5" onClick={() => setAddRowOpen(true)}>
                    <Plus className="h-3 w-3" /> Add Row
                  </Button>
                )}
                {schemaData && (
                  <button
                    className="p-1 rounded hover:bg-accent text-muted-foreground"
                    onClick={() => setShowSchema((s) => !s)}
                  >
                    <Info className="h-3.5 w-3.5" />
                  </button>
                )}
                {showSchema && schemaData && <SchemaPopover columns={schemaData.columns} />}
              </div>
            </div>

            <div className="flex-1 min-h-0 overflow-hidden">
              {rowsLoading ? (
                <div className="flex items-center justify-center h-24">
                  <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                </div>
              ) : (
                <ResultsTable
                  rows={rowsData?.rows ?? []}
                  columns={schemaData?.columns}
                  onEdit={pkCol ? (row) => setEditRow(row) : undefined}
                  onDelete={pkCol ? (row) => setDeleteRow(row) : undefined}
                />
              )}
            </div>

            {rowsData && rowsData.pages > 1 && (
              <div className="flex items-center justify-between px-4 py-2 border-t border-border shrink-0 text-xs text-muted-foreground">
                <span>Page {rowsData.page} of {rowsData.pages}</span>
                <div className="flex items-center gap-1">
                  <button disabled={page <= 1} onClick={() => setPage((p) => p - 1)} className="p-1 rounded hover:bg-accent disabled:opacity-40 disabled:cursor-not-allowed">
                    <ChevronLeft className="h-3.5 w-3.5" />
                  </button>
                  <button disabled={page >= rowsData.pages} onClick={() => setPage((p) => p + 1)} className="p-1 rounded hover:bg-accent disabled:opacity-40 disabled:cursor-not-allowed">
                    <ChevronRight className="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {/* Modals */}
      {showCreateTable && (
        <CreateTableModal
          onClose={() => { setShowCreateTable(false); createTable.reset(); }}
          onSubmit={(name, columns) => createTable.mutate({ name, columns })}
          isPending={createTable.isPending}
          error={createTable.error instanceof Error ? createTable.error.message : null}
        />
      )}

      {addRowOpen && schemaData && (
        <RowFormModal
          mode="add"
          table={selectedTable!}
          columns={schemaData.columns}
          onClose={() => { setAddRowOpen(false); insertRow.reset(); }}
          onSubmit={(values) => insertRow.mutate(values)}
          isPending={insertRow.isPending}
          error={insertRow.error instanceof Error ? insertRow.error.message : null}
        />
      )}

      {editRow && schemaData && (
        <RowFormModal
          mode="edit"
          table={selectedTable!}
          columns={schemaData.columns}
          initialValues={editRow}
          onClose={() => { setEditRow(null); updateRow.reset(); }}
          onSubmit={handleEditSubmit}
          isPending={updateRow.isPending}
          error={updateRow.error instanceof Error ? updateRow.error.message : null}
        />
      )}

      {deleteRow && (
        <ConfirmDeleteDialog
          onConfirm={handleDeleteConfirm}
          onCancel={() => { setDeleteRow(null); deleteMutation.reset(); }}
          isPending={deleteMutation.isPending}
        />
      )}
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
      <div className="shrink-0 border-b border-border">
        <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border/50 bg-muted/30">
          <span className="text-xs text-muted-foreground font-medium">SQL Editor</span>
          <span className="text-xs text-muted-foreground/50">— SELECT only · Ctrl+Enter to run</span>
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
            value={sql}
            onChange={(e) => setSql(e.target.value)}
            onKeyDown={handleKeyDown}
            spellCheck={false}
            rows={6}
            placeholder={"SELECT * FROM agents LIMIT 20"}
            className="w-full px-4 py-3 font-mono text-sm bg-transparent text-foreground placeholder:text-muted-foreground/40 resize-none focus:outline-none"
          />
          <div className="absolute bottom-2 right-3 flex items-center gap-2">
            <Button size="sm" onClick={runQuery} disabled={!sql.trim() || query.isPending} className="gap-1.5">
              {query.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Play className="h-3.5 w-3.5" />}
              Run
            </Button>
          </div>
        </div>
      </div>

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
              <span className="text-xs text-muted-foreground">{query.data.elapsedMs}ms</span>
              {query.data.capped && (
                <span className="text-xs text-amber-500 flex items-center gap-1">
                  <AlertCircle className="h-3 w-3" /> Capped at 500 rows
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
                tab === t ? "bg-accent text-foreground" : "text-muted-foreground hover:text-foreground hover:bg-accent/50"
              )}
            >
              {t === "tables" ? "Table Browser" : "Query Runner"}
            </button>
          ))}
        </div>
        <div className="ml-auto flex items-center gap-1.5">
          {tab === "query" && (
            <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium bg-amber-500/10 text-amber-600 border border-amber-500/20">
              read-only
            </span>
          )}
        </div>
      </div>

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
