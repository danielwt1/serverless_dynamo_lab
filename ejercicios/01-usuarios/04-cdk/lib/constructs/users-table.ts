import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import { Construct } from 'constructs';
import type { UsersLabConfiguration } from '../config/users-config';

/** Propiedades del componente de persistencia del dominio Usuarios. */
export interface UsersTableProps {
  readonly configuration: UsersLabConfiguration['table'];
}

/**
 * Componente que crea el almacenamiento según los patrones de acceso del
 * dominio Usuarios.
 *
 * Extender `Construct` permite tratar esta pieza como una unidad: el stack la
 * compone, mientras que esta clase conserva los detalles de DynamoDB.
 * GSI1 es disperso: solo usuarios activos tienen sus atributos de índice y una
 * Query evita un Scan. La proyección cubre el listado sin lecturas adicionales.
 */
export class UsersTable extends Construct {
  /** Tabla expuesta para que otros componentes declaren dependencias seguras. */
  public readonly table: dynamodb.Table;

  public constructor(scope: Construct, id: string, props: UsersTableProps) {
    super(scope, id);

    const { configuration } = props;
    this.table = new dynamodb.Table(this, 'Resource', {
      // PK y SK permiten varios tipos de ítem y relaciones en una sola tabla.
      partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING },
      // No se fija capacidad: DynamoDB cobra por solicitud en este laboratorio.
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      // DESTROY hace que la tabla desaparezca con cdk destroy en este lab.
      removalPolicy: configuration.removalPolicy,
      // Desactivado solo para permitir la limpieza completa del laboratorio.
      deletionProtection: configuration.deletionProtection,
      pointInTimeRecoverySpecification: {
        // PITR se deja apagado por ser efímero; producción debería activarlo.
        pointInTimeRecoveryEnabled: configuration.pointInTimeRecoveryEnabled
      }
    });

    this.table.addGlobalSecondaryIndex({
      // Índice para consultar usuarios activos sin hacer Scan sobre la tabla.
      indexName: configuration.activeUsersIndexName,
      partitionKey: { name: 'GSI1PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'GSI1SK', type: dynamodb.AttributeType.STRING },
      // Incluye solo campos del listado y evita una lectura adicional por ítem.
      projectionType: dynamodb.ProjectionType.INCLUDE,
      nonKeyAttributes: ['userId', 'name', 'lastName', 'email', 'state', 'created_at']
    });
  }
}
