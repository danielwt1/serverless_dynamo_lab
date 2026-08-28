# follower-fanout-worker

Código de la Lambda que recorre followers y encola batches de notificación.
Su algoritmo, checkpoint, reintentos, configuración e idempotencia están en
las guías de [Lambdas](../../../documentacion/02-implementacion-lambdas.md),
[eventos](../../../documentacion/03-eventos-fanout-y-resiliencia.md) e
[infraestructura](../../../documentacion/04-infraestructura-aws.md).
