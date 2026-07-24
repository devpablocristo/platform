# Access management

Feature reutilizable para administrar usuarios, tenants e invitaciones desde
una aplicación SaaS.

La implementación TypeScript es agnóstica del proveedor de identidad y del
transporte HTTP. El consumer inyecta un `AccessManagementClient` y el contexto
de acceso autenticado. Las invitaciones siempre usan el tenant activo: el
módulo no ofrece un selector alternativo de tenant.
