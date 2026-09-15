# 1. Modelo DynamoDB y patrones de acceso

## Consultas que determinan el diseño

| Necesidad | Operación | Clave |
|---|---|---|
| Autenticar por correo | `GetItem` | `PK=EMAIL#<email-normalizado>`, `SK=UNIQUE` |
| Listar activos recientes | `Query` a `GSI1` | `GSI1PK=ACTIVE_USERS`, `GSI1SK=<fecha>#USER#<id>` |
| Obtener perfil por id | `GetItem` | `PK=USER#<id>`, `SK=PROFILE` |

La tabla se llama `users`, con PK/SK de tipo String y GSI `GSI1`. No se necesita `Scan`: las tres lecturas conocen su clave de partición.

## Items escritos al crear un usuario

```text
# Reserva de unicidad / lookup de login
PK       = EMAIL#ana@example.com
SK       = UNIQUE
userId   = <uuid>
password = <hash: pendiente en el código actual>

# Perfil
PK         = USER#<uuid>
SK         = PROFILE
email      = ana@example.com
state      = ACTIVE
GSI1PK     = ACTIVE_USERS
GSI1SK     = 2026-08-24T15:00:00Z#USER#<uuid>
created_at = 2026-08-24T15:00:00Z
```

`GSI1` es disperso: un usuario inactivo no debe llevar `GSI1PK/GSI1SK`; así desaparece de la vista operativa sin borrar su perfil.

## Unicidad atómica del correo

`create-user` usa `TransactWriteItems`: reserva el correo con un `Put` condicional y crea el perfil en la misma transacción. Si el correo ya existe, la condición falla y no se escribe nada del perfil nuevo. Esto evita la carrera de “consultar primero y escribir después”.

Normalizar el correo (`trim` + minúsculas) antes de construir la clave evita tratar `ANA@Ejemplo.com` y `ana@ejemplo.com` como identidades distintas.

## Paginación y consistencia

La lista activa usa `LastEvaluatedKey` como cursor Base64 opaco y orden descendente por fecha. Un GSI es eventualmente consistente: después de crear o cambiar estado, la lista puede tardar brevemente en reflejarlo. La autenticación usa `GetItem` con consistencia fuerte sobre la reserva de email.

## Deuda visible de seguridad

El adaptador actual guarda el literal `"PASSWORD"`, y la autenticación compara strings. Eso sirve solo como esqueleto didáctico de acceso; no es aceptable para producción. Antes de desplegar usuarios reales, hashear con Argon2id/bcrypt, guardar solo el hash, usar comparación segura, cifrado en reposo y no registrar secretos en logs.
