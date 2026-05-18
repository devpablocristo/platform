import type { CrudPageConfig } from "./types";
import { withDefaultCrudFeatureFlags } from "./crudFeatureDefaults";

/**
 * Fusión “canónica” previa a preferencias de usuario: defaults de plantilla del módulo CRUD.
 * El `resourceId` queda reservado para futuras ramas por recurso (p. ej. columnas por vertical).
 */
export function mergeCanonicalCrudDefaults<T extends { id: string }>(
  _resourceId: string,
  config: CrudPageConfig<T>,
): CrudPageConfig<T> {
  return withDefaultCrudFeatureFlags(config);
}
