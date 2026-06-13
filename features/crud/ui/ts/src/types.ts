/**
 * Tipos del módulo CRUD (TypeScript): columnas, formulario, dataSource y acciones.
 * Sin tipos de dominio de negocio; la fila solo requiere `id: string`.
 */
import type { ReactNode } from "react";

/** Contexto para slots opcionales del listado (p. ej. filtros en cabecera). */
export type CrudListHeaderSlotContext<T> = { items: T[] };

export type CrudFieldValue = string | boolean;
export type CrudFormValues = Record<string, CrudFieldValue>;
export type CrudLifecycleView = "active" | "archived" | "trash";

export type CrudColumn<T> = {
  key: keyof T & string;
  header: string;
  render?: (value: unknown, row: T) => ReactNode;
  className?: string;
  /**
   * Si es false, la columna no muestra ordenación en cabecera. Por defecto true.
   */
  sortable?: boolean;
  /**
   * Valor usado para ordenar filas completas (si no, se usa `row[key]`).
   */
  sortValue?: (row: T) => unknown;
};

export type CrudFormField = {
  key: string;
  label: string;
  type?: "text" | "email" | "tel" | "number" | "date" | "datetime-local" | "textarea" | "select" | "checkbox";
  placeholder?: string;
  required?: boolean;
  rows?: number;
  fullWidth?: boolean;
  createOnly?: boolean;
  editOnly?: boolean;
  options?: Array<{ label: string; value: string }>;
};

export type CrudDataSource<T extends { id: string }> = {
  list?: (params: { view: CrudLifecycleView }) => Promise<T[]>;
  create?: (values: CrudFormValues) => Promise<unknown>;
  update?: (row: T, values: CrudFormValues) => Promise<unknown>;
  archive?: (row: T) => Promise<unknown>;
  trash?: (row: T) => Promise<unknown>;
  unarchive?: (row: T) => Promise<unknown>;
  restore?: (row: T) => Promise<unknown>;
  purge?: (row: T) => Promise<unknown>;
};

export type CrudHelpers<T extends { id: string }> = {
  items: T[];
  reload: () => Promise<void>;
  setError: (message: string) => void;
};

export type CrudToolbarAction<T extends { id: string }> = {
  id: string;
  label: string;
  kind?: "primary" | "secondary" | "danger" | "success";
  isVisible?: (ctx: { view: CrudLifecycleView; items: T[] }) => boolean;
  onClick: (helpers: CrudHelpers<T>) => Promise<void> | void;
};

export type CrudRowAction<T extends { id: string }> = {
  id: string;
  label: string;
  kind?: "primary" | "secondary" | "danger" | "success";
  isVisible?: (row: T, ctx: { view: CrudLifecycleView }) => boolean;
  onClick: (row: T, helpers: CrudHelpers<T>) => Promise<void> | void;
};

/**
 * Cliente HTTP mínimo cuando se usa `basePath` sin `dataSource.list`.
 */
export type CrudHttpClient = {
  json<TResponse>(path: string, init?: { method?: string; body?: Record<string, unknown> }): Promise<TResponse>;
};

/** Modos de vista conocidos por el shell CRUD (la app puede declarar solo un subconjunto). */
export type CrudViewModeId = "list" | "gallery" | "kanban";

export type CrudViewModeConfig = {
  id: CrudViewModeId;
  label: string;
  path: string;
  ariaLabel?: string;
  isDefault?: boolean;
};

/** Flags opcionales del template de listado (producto / preferencias de usuario pueden apagarlos). */
export type CrudFeatureFlags = {
  creatorFilter?: boolean;
  /**
   * Franja bajo el título con filtros rápidos tipo chip (p. ej. ownership vía `listHeaderInlineSlot`).
   * Si es `false`, no se renderiza aunque exista slot (p. ej. inventario sin chips hasta definir otro filtro).
   */
  headerQuickFilterStrip?: boolean;
  csvToolbar?: boolean;
  /** Barra de búsqueda interna del listado. Por defecto true. `false` oculta el input de search. */
  searchBar?: boolean;
  /** Botón de paginación «Cargar más» al pie del listado. Por defecto true. `false` lo oculta. */
  pagination?: boolean;
  /** Botón «Ver archivados» en toolbar. `false` equivale a `supportsArchived: false` al aplicar override. */
  archivedToggle?: boolean;
  /** Botón «Ver papelera» en toolbar. `false` equivale a `supportsTrash: false` al aplicar override. */
  trashToggle?: boolean;
  /** Botón «Crear» en toolbar. `false` equivale a `allowCreate: false` al aplicar override. */
  createAction?: boolean;
  tagsColumn?: boolean;
  /** Cabeceras clicables para ordenar filas (asc/desc). Por defecto true. */
  columnSort?: boolean;
};

/**
 * Configuración de la página: etiquetas ya resueltas en el idioma de la app.
 */
export type CrudPageConfig<T extends { id: string }> = {
  basePath?: string;
  /** Query sin `?`; se concatena al GET de listado. */
  listQuery?: string;
  /** Nombre del parámetro de cursor. Por defecto `cursor`. */
  paginationCursorParam?: "cursor" | "after" | (string & {});
  dataSource?: CrudDataSource<T>;
  /** Obligatorio si hay `basePath` y no hay `dataSource.list`. */
  httpClient?: CrudHttpClient;
  supportsArchived?: boolean;
  supportsTrash?: boolean;
  allowCreate?: boolean;
  allowEdit?: boolean;
  allowArchive?: boolean;
  allowTrash?: boolean;
  allowUnarchive?: boolean;
  allowRestore?: boolean;
  allowPurge?: boolean;
  label: string;
  labelPlural: string;
  labelPluralCap: string;
  columns: CrudColumn<T>[];
  /** Variante opcional de columnas para la vista de archivados. */
  archivedColumns?: CrudColumn<T>[];
  /** Variante opcional de columnas para la vista de papelera. */
  trashColumns?: CrudColumn<T>[];
  formFields: CrudFormField[];
  searchText: (row: T) => string;
  toFormValues: (row: T) => CrudFormValues;
  toBody?: (values: CrudFormValues) => Record<string, unknown>;
  isValid: (values: CrudFormValues) => boolean;
  searchPlaceholder?: string;
  emptyState?: string;
  archivedEmptyState?: string;
  trashEmptyState?: string;
  createLabel?: string;
  toolbarActions?: CrudToolbarAction<T>[];
  rowActions?: CrudRowAction<T>[];
  /** i18n opcional para textos de campos (por defecto identidad). */
  formatFieldText?: (raw: string) => string;
  /** Títulos (por defecto identidad). */
  sentenceCase?: (s: string) => string;
  /**
   * Si está definido, el botón Editar de la fila invoca esto en lugar de abrir el formulario inline.
   * Útil cuando el producto tiene un editor dedicado (modal o ruta).
   */
  onExternalEdit?: (row: T) => void;
  /**
   * Clic en la fila (fuera de la columna de acciones). Útil cuando no hay columna «Acciones» y el detalle
   * se abre en modal u otra superficie.
   */
  onRowClick?: (row: T) => void;
  /**
   * Filtra filas en cliente después del listado y antes del input de búsqueda.
   */
  preSearchFilter?: (items: T[]) => T[];
  /**
   * Contenido entre la búsqueda y la fila de botones de cabecera (p. ej. filtros tipo píldora).
   */
  listHeaderInlineSlot?: (ctx: CrudListHeaderSlotContext<T>) => ReactNode;
  /**
   * Búsqueda controlada desde afuera (p. ej. un buscador global de la página).
   * Si está definido, oculta el input de búsqueda interno y usa este valor.
   */
  externalSearch?: string;
  /** Modos de vista alternativos a la lista (galería, kanban, etc.) — el producto define rutas y render. */
  viewModes?: CrudViewModeConfig[];
  /** Interruptores de plantilla del listado; por defecto todo encendido vía `mergeCanonicalCrudDefaults`. */
  featureFlags?: CrudFeatureFlags;
  /** Vista inicial del listado. */
  initialView?: CrudLifecycleView;
};
