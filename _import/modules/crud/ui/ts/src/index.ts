/**
 * CRUD de consola (React). Primitivas de layout en `@devpablocristo/core-browser/crud`.
 */
export { CrudPage, type CrudPageProps } from "./CrudPage";
export { CrudShellHeaderActionsColumn, type CrudShellHeaderActionsColumnProps, type CrudShellSearchFieldProps } from "./CrudShellHeaderActionsColumn";
export { CrudUiPreferencesPanel, type CrudUiPreferencesPanelCopy, type CrudUiPreferencesPanelProps, type CrudUiPreferencesResource } from "./CrudUiPreferencesPanel";
export {
  CRUD_FEATURE_FLAGS_ALL_ON,
  withDefaultCrudFeatureFlags,
} from "./crudFeatureDefaults";
export { mergeCanonicalCrudDefaults } from "./surface";
export {
  CRUD_UI_PREFERENCES_FEATURE_KEYS,
  createCrudUiPreferencesApi,
  type CrudUiResourceOverride,
  type CreateCrudUiPreferencesApiOptions,
} from "./crudUiPreferences";
export type {
  CrudColumn,
  CrudDataSource,
  CrudFeatureFlags,
  CrudFieldValue,
  CrudFormField,
  CrudFormValues,
  CrudHelpers,
  CrudHttpClient,
  CrudListHeaderSlotContext,
  CrudPageConfig,
  CrudRowAction,
  CrudToolbarAction,
  CrudViewModeConfig,
  CrudViewModeId,
} from "./types";
export {
  crudStringsEs,
  defaultCrudStrings,
  interpolate,
  mergeCrudStrings,
  type CrudStrings,
} from "./strings";
export { CrudPathSegment, crudItemPath, crudListPath } from "./restPaths";
export {
  INTERNAL_FAVORITE_FIELD_KEY,
  INTERNAL_FAVORITE_LABEL,
  INTERNAL_NOTES_FIELD_KEY,
  INTERNAL_NOTES_LABEL,
  INTERNAL_TAGS_FIELD_KEY,
  INTERNAL_TAGS_LABEL,
  buildInternalFavoriteField,
  buildInternalNotesField,
  buildInternalTagsField,
  buildStandardInternalFields,
  formatTagCsv,
  parseTagCsv,
} from "./entityFields";
export {
  buildCsvToolbarActions,
  mergeCsvToolbarConfig,
  type CrudCsvServerExportPort,
  type CrudCsvServerImportPort,
  type CrudCsvToolbarUiPort,
  type CsvServerImportPreview,
  type CsvServerImportResult,
  type CsvToolbarMergeMode,
  type CsvToolbarMessages,
  type MergeCsvToolbarParams,
} from "./csvToolbarMerge";
