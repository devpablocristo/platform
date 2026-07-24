# `@devpablocristo/platform-access-management`

Módulo React para la administración SaaS de usuarios, tenants e invitaciones.
El consumer conserva el transporte, la autorización y el vocabulario de roles;
la feature aporta la interfaz y los contratos comunes.

## Garantías

- email obligatorio y username nullable;
- contraseña mínima de 8 caracteres, sin normalizar ni recortar;
- control accesible para mostrar u ocultar la contraseña;
- un único buscador para las tres pestañas;
- invitaciones creadas y listadas exclusivamente para `activeTenant`;
- respuestas tardías de un tenant anterior se descartan;
- al cambiar usuario o tenant se cierra cualquier formulario abierto;
- sin selector de tenant dentro de Invitaciones.

## Uso

```tsx
import { AccessManagement } from "@devpablocristo/platform-access-management";
import "@devpablocristo/platform-access-management/styles.css";

<AccessManagement
  client={accessClient}
  context={{
    userId: session.userId,
    activeTenant: session.activeTenant,
    manageableTenants: session.manageableTenants,
    policy: {
      canManageUsers: session.isPlatformOperator,
      canManageTenants: session.isPlatformOwner,
      canManageInvitations: session.canManageActiveTenant,
    },
  }}
/>;
```

`AccessManagementClient` es deliberadamente provider-neutral. Un adapter puede
usar Clerk, Firebase, un BFF propio u otro proveedor sin filtrar esos detalles
al componente.
