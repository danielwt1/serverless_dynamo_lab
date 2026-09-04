# 01. Usuarios de una aplicacion

## Contexto
Una aplicacion de aprendizaje necesita registrar personas, permitirles iniciar sesion y mostrar a operaciones las cuentas que siguen activas.

## Problema
El equipo no quiere recorrer todos los usuarios para iniciar sesion. Tampoco puede aceptar dos cuentas con el mismo correo. Una cuenta desactivada debe dejar de aparecer en la vista operativa.

## Reglas
- El correo pertenece a una sola cuenta.
- Crear una cuenta repetida no puede sobreescribir la existente.
- Una cuenta desactivada conserva su historial.
- No se almacenan passwords en texto plano.

## Preguntas que debes resolver
- Que consultas exactas necesita el login y op  eraciones?
- Que datos deben estar juntos y cuales necesitan otra vista?
- Como garantizas unicidad bajo solicitudes concurrentes?
- Como paginas una lista grande de usuarios activos?

## Entregables
Documento de patrones de acceso, modelo de tabla, ejemplos de items, contrato de `CreateUser` y `GetUser`, y pruebas para correo duplicado y usuario inexistente.

## Documentación del laboratorio

Sigue la [guía central](documentacion/README.md). Allí están el modelo
DynamoDB, cada Lambda, el contrato HTTP, la infraestructura y el despliegue.
Las carpetas de fase conservan el código y solo enlazan a esa fuente de verdad.
