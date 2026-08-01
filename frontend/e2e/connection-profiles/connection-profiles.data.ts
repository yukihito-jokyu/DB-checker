export type DatabaseType = "mysql" | "postgres";

export type ConnectionProfileData = {
  name: string;
  dbType: DatabaseType;
  host: string;
  port: number;
  database: string;
  schema?: string;
  user: string;
};

export const postgresProfile: ConnectionProfileData = {
  name: "E2E PostgreSQL",
  dbType: "postgres",
  host: "postgres.e2e.invalid",
  port: 5432,
  database: "e2e_postgres",
  schema: "e2e_schema",
  user: "e2e_user",
};

export const mysqlProfile: ConnectionProfileData = {
  name: "E2E MySQL",
  dbType: "mysql",
  host: "mysql.e2e.invalid",
  port: 3306,
  database: "e2e_mysql",
  user: "e2e_user",
};

export const updatedMySQLProfile: ConnectionProfileData = {
  name: "E2E MySQL 更新後",
  dbType: "mysql",
  host: "mysql-updated.e2e.invalid",
  port: 3307,
  database: "e2e_mysql_updated",
  user: "e2e_updated_user",
};
