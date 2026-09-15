# 3. Contrato HTTP

La fuente de verdad del contrato es [`03-api/contracts/openapi.yaml`](../03-api/contracts/openapi.yaml).

| Ruta | Método | Lambda | Resultado |
|---|---|---|---|
| `/users` | POST | create-user | 201 o conflicto 409 |
| `/auth/login` | POST | authenticate-user | userId al validar credenciales |
| `/users/active` | GET | list-active-users | página ordenada y `nextCursor` |

## Reglas del contrato

- Password es `writeOnly`: nunca debe aparecer en respuestas.
- `limit` está entre 1 y 100, con valor por defecto 10.
- `cursor` es opaco: el cliente no conoce ni modifica la clave DynamoDB.
- Los errores deben tener código y mensaje; no exponer detalles del SDK, tablas ni secretos.

## Alineación pendiente

OpenAPI indica 401 para credenciales inválidas e incluye una respuesta 403 para usuario inactivo. El handler actual responde 404 y aún no consulta/valida estado. La guía representa el contrato objetivo; antes de desplegar API pública hay que alinear implementación y OpenAPI.
