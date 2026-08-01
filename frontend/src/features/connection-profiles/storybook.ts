import type { ConnectionProfile, ConnectionProfiles } from "./types";

export const postgresProfile = {
  id: "postgres-local",
  name: "PostgreSQL ローカル",
  dbType: "postgres",
  host: "127.0.0.1",
  port: 5432,
  database: "db_checker",
  schema: "public",
  user: "db_checker",
} satisfies ConnectionProfile;

export const mysqlProfile = {
  id: "mysql-staging",
  name: "MySQL ステージング",
  dbType: "mysql",
  host: "mysql.staging.example.com",
  port: 3306,
  database: "app",
  schema: "",
  user: "viewer",
} satisfies ConnectionProfile;

export const longProfile = {
  id: "postgres-reporting",
  name: "分析チーム向けステージング環境読み取り専用PostgreSQL接続プロファイル",
  dbType: "postgres",
  host: "reporting-database.staging.internal.example.com",
  port: 5432,
  database: "application_reporting_database",
  schema: "business_intelligence",
  user: "readonly_analytics_application_user",
} satisfies ConnectionProfile;

export const defaultProfiles = {
  profiles: [postgresProfile, mysqlProfile],
  activeProfileId: postgresProfile.id,
} satisfies ConnectionProfiles;

export const activatedProfiles = {
  profiles: [postgresProfile, mysqlProfile],
  activeProfileId: mysqlProfile.id,
} satisfies ConnectionProfiles;
