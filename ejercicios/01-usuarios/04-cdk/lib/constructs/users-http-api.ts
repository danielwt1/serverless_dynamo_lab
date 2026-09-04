import * as cdk from 'aws-cdk-lib';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import * as apigatewayv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as integrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as logs from 'aws-cdk-lib/aws-logs';
import { Construct } from 'constructs';
import type { UsersLabConfiguration, UsersOperation } from '../config/users-config';

/** Una operación de negocio ya conectada a la Lambda que la implementa. */
export interface OperationWithFunction {
  readonly operation: UsersOperation;
  readonly fn: cdk.aws_lambda.IFunction;
}

/** Propiedades del componente de entrada HTTP de Usuarios. */
export interface UsersHttpApiProps {
  readonly operationsWithFunctions: readonly OperationWithFunction[];
  readonly configuration: UsersLabConfiguration['api'];
}

/**
 * Componente que expone las operaciones mediante HTTP API.
 *
 * Usa constructos L2 de CDK: estos crean integración, ruta y permiso de
 * invocación de Lambda de forma coherente. Cada integración restringe dicho
 * permiso al path de su ruta por defecto.
 *
 * La API es pública porque es un laboratorio. Antes de exponerla se debe añadir
 * un authorizer y límites de tráfico; CORS no es autenticación.
 */
export class UsersHttpApi extends Construct {
  /** API expuesta para publicar su endpoint o añadir dominios en otro componente. */
  public readonly api: apigatewayv2.HttpApi;

  public constructor(scope: Construct, id: string, props: UsersHttpApiProps) {
    super(scope, id);

    const { operationsWithFunctions, configuration } = props;
    // Permite obtener datos del stack actual, como su nombre. No consulta AWS.
    const stack = cdk.Stack.of(this);
    const apiLogs = new logs.LogGroup(this, 'AccessLogs', {
      // Nombre explícito: facilita localizar los access logs en CloudWatch.
      logGroupName: `/aws/apigateway/${stack.stackName}-${configuration.name}`,
      // Días que CloudWatch conserva cada evento antes de borrarlo.
      retention: configuration.logRetention,
      // En este lab, cdk destroy también elimina este grupo y sus eventos.
      removalPolicy: configuration.logRemovalPolicy
    });
    // Se desactiva el stage implícito porque abajo se crea uno explícito con
    // logs y autoDeploy. Así evitamos tener dos stages por accidente.
    this.api = new apigatewayv2.HttpApi(this, 'Resource', {
      // Nombre visible de la API en la consola de API Gateway.
      apiName: configuration.name,
      createDefaultStage: false
    });

    for (const { operation, fn } of operationsWithFunctions) {
      new apigatewayv2.HttpRoute(this, `${operation.id}Route`, {
        // Esta ruta pertenece a la API creada unas líneas arriba.
        httpApi: this.api,
        // Combina método y path: por ejemplo, POST + /users.
        routeKey: apigatewayv2.HttpRouteKey.with(operation.route.path, toHttpMethod(operation.route.method)),
        // La integración es el puente: al coincidir la ruta, API Gateway
        // invoca esta Lambda y devuelve su respuesta al cliente HTTP.
        integration: new integrations.HttpLambdaIntegration(`${operation.id}Integration`, fn, {
          // Define la forma del evento que recibe el handler Go. Debe coincidir
          // con el tipo de evento HTTP API v2 que interpreta la Lambda.
          payloadFormatVersion: apigatewayv2.PayloadFormatVersion.VERSION_2_0,
          // Máxima espera de API Gateway por la respuesta de Lambda. Son 9 s,
          // un segundo menos que el timeout de la Lambda (10 s) para que API
          // Gateway pueda responder un 504 sin esperar el corte de Lambda.
          timeout: cdk.Duration.millis(configuration.integrationTimeoutMilliseconds),
          // CDK crea AWS::Lambda::Permission. `true` lo limita al path de esta
          // ruta; con `false` permitiría invocación desde cualquier ruta de la API.
          scopePermissionToRoute: true
        })
      });
    }

    new apigatewayv2.HttpStage(this, 'DefaultStage', {
      // Publica las rutas e integraciones de `this.api`.
      httpApi: this.api,
      // `$default` evita un prefijo como /dev en la URL pública.
      stageName: '$default',
      // Solo después de `cdk deploy`: cada cambio de configuración de la API
      // que llegue a AWS se publica en este stage sin crear un deployment manual.
      autoDeploy: true,
      accessLogSettings: {
        // Destino de los access logs: el Log Group creado al inicio de esta clase.
        destination: new apigatewayv2.LogGroupLogDestination(apiLogs),
        // Una línea JSON por solicitud. No registra body, headers ni contraseñas.
        format: apigateway.AccessLogFormat.custom(JSON.stringify({
          // Identificador para localizar una solicitud concreta en CloudWatch.
          requestId: '$context.requestId',
          // Código HTTP final, por ejemplo 200, 400, 401 o 504.
          status: '$context.status',
          // Ruta que atendió API Gateway, por ejemplo POST /users.
          routeKey: '$context.routeKey',
          // Mensaje técnico si API Gateway no pudo completar la integración Lambda.
          integrationError: '$context.integrationErrorMessage'
        }))
      }
    });
  }
}

/** Traduce el catálogo propio a la enumeración tipada que entiende CDK. */
function toHttpMethod(method: UsersOperation['route']['method']): apigatewayv2.HttpMethod {
  return method === 'GET' ? apigatewayv2.HttpMethod.GET : apigatewayv2.HttpMethod.POST;
}
