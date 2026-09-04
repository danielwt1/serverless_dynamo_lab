# 15. Catálogo de lectura intensiva

## Contexto

Un catálogo de precios recibe muchas más lecturas que actualizaciones. La página principal tolera una pequeña demora en reflejar un precio nuevo, pero checkout necesita conocer un precio confirmado.

## Problema

Decide si DynamoDB sin caché, DAX, una proyección o una combinación resuelve cada lectura. El objetivo no es instalar una caché por defecto: debes medir la necesidad, entender la consistencia y demostrar la invalidación o expiración elegida.

## Reglas

- La lectura de catálogo puede privilegiar latencia y costo.
- La validación de precio de checkout no puede aceptar información obsoleta sin una política explícita.
- La caché no es la fuente de verdad.
- Un fallo o dato vencido en caché debe degradar de forma segura.

## Criterios de aceptación

- Clasifica cada access pattern por frescura, latencia, volumen y tolerancia a dato obsoleto.
- Compara costo/operación de lectura directa, DAX y una alternativa de caché.
- Demuestra qué sucede inmediatamente después de actualizar un precio.
- Define métricas y alarmas que justificarían adoptar o retirar la caché.

## Alcance de servicios

DynamoDB, Lambda, API Gateway y DAX únicamente si la propuesta demuestra que aporta valor. Puede realizarse con una simulación de carga para evitar costos innecesarios.

## Nivel de ayuda

Decisión de arquitectura: no hay una respuesta obligatoria que incluya DAX. Se evalúa la evidencia detrás de usarlo o descartarlo.
