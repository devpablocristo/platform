import type { CrudFeatureFlags, CrudPageConfig } from "./types";

/** Plantilla: todos los flags de listado encendidos salvo que el recurso los sobrescriba. */
export const CRUD_FEATURE_FLAGS_ALL_ON: CrudFeatureFlags = {
  creatorFilter: true,
  headerQuickFilterStrip: true,
  csvToolbar: true,
  pagination: true,
  tagsColumn: true,
  columnSort: true,
};

export function withDefaultCrudFeatureFlags<T extends { id: string }>(config: CrudPageConfig<T>): CrudPageConfig<T> {
  return {
    ...config,
    featureFlags: {
      ...CRUD_FEATURE_FLAGS_ALL_ON,
      ...(config.featureFlags ?? {}),
    },
  };
}
