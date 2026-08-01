import type { wails } from "@wails/go/models";

export type ConnectionProfile = wails.ProfileResponse;

export type ConnectionProfileDraft = {
  id: string;
  name: string;
  dbType: "postgres" | "mysql";
  host: string;
  port: number;
  database: string;
  schema?: string;
  user: string;
  password: string;
};

export type ConnectionProfiles = {
  profiles: ConnectionProfile[];
  activeProfileId?: string;
};

export type LoadState = "loading" | "success" | "error";
export type OperationState = "saving" | "activating" | "deleting" | null;
