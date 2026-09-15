# Validación contra `Model Dynamo Task Expired.xlsx`

| Patrón | Implementación |
| --- | --- |
| AP-01 crear tarea | Transacción con item principal, proyección PENDING y GSI por uno de seis shards. |
| AP-02 completar | Lectura fuerte y transacción que cambia a `COMPLETED`, elimina GSI y proyección. |
| AP-03 listar todas | Query por `OWNER_ID#`, prefijo `TASK_ID#`, páginas de 10. |
| AP-04 listar pendientes | Query por `OWNER_ID#`, prefijo `STATUS#PENDING#`, páginas de 10. |
| AP-05 buscar vencidas | Seis Query al GSI `task_gsi_pending`, sin Scan y con límite temporal fijo. |
| AP-06 iniciar/retomar | Batch abierto identificado por GSI y creación condicional de META + checkpoints. |
| AP-07 checkpoint | Una página por vez; el cursor solo se confirma después de reservar sus eventos. |
| AP-08 cerrar | Pasa por `PUBLISHING`, vacía el outbox y termina en `COMPLETED`. |

El XLSX aprobado describe una versión intermedia “sin SNS”. El README exige publicar `TaskOverdue`, por lo que la implementación extiende AP-07/AP-08 con outbox, SNS y consumidor idempotente. También usa un META por corrida y un GSI disperso de procesos abiertos, que evita que un batch fallido quede oculto al siguiente trigger.

La forma canónica del shard es `SHARD#00#STATUS#PENDING` a `SHARD#05#STATUS#PENDING`; corrige la variante tipográfica `SHARD-(0-5)` de una celda del XLSX y coincide con sus simulaciones de Query.
