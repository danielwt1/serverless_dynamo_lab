# Automatización para AWS

Esta carpeta es exclusiva para AWS real. Ejecútala desde la raíz del laboratorio:

```bash
./scripts/aws/01-create-resources.sh
./scripts/aws/02-build-and-push-images.sh
./scripts/aws/03-deploy-lambdas-and-api.sh
./scripts/aws/04-seed-followers.sh # opcional: inserta 250 followers demo
```

El seed de esta carpeta es una acción explícita sobre AWS, separada del seed local. La guía de [despliegue y verificación](../../documentacion/05-despliegue-y-verificacion.md) conserva el camino equivalente por consola.

La autenticación no es un argumento del script: AWS CLI resuelve la sesión ya iniciada y el perfil activo. No se reciben ni guardan access keys, secretos o tokens. Todo script futuro debe empezar verificando `aws sts get-caller-identity`, mostrar cuenta, rol y región destino, y pedir confirmación antes de modificar recursos. Solo parámetros no sensibles, como entorno o nombre de función, pueden configurarse explícitamente.
