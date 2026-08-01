import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

import type { ConnectionProfile, OperationState } from "../types";

type ConnectionProfileListProps = {
  profiles: ConnectionProfile[];
  activeProfileId?: string;
  actionsDisabled: boolean;
  operation: OperationState;
  onActivate: (profileId: string) => void;
  onEdit: (profile: ConnectionProfile, trigger: HTMLButtonElement) => void;
  onDelete: (profile: ConnectionProfile, trigger: HTMLButtonElement) => void;
};

export function ConnectionProfileList({
  profiles,
  activeProfileId,
  actionsDisabled,
  operation,
  onActivate,
  onEdit,
  onDelete,
}: ConnectionProfileListProps) {
  return (
    <ul className="mt-4 grid gap-4" aria-label="接続プロファイル一覧">
      {profiles.map((profile) => {
        const isActive = activeProfileId === profile.id;
        return (
          <li key={profile.id}>
            <Card>
              <CardHeader className="gap-2 p-5 pb-2">
                <div className="flex flex-wrap items-center gap-2">
                  <CardTitle className="break-words">{profile.name}</CardTitle>
                  {isActive ? (
                    <Badge className="border-primary/20 bg-primary/10 text-primary hover:text-white">
                      使用中
                    </Badge>
                  ) : null}
                </div>
              </CardHeader>
              <CardContent className="flex flex-col gap-1 px-5 pb-0 text-base text-muted-foreground">
                <p className="break-words">
                  {profile.dbType === "mysql" ? "MySQL" : "PostgreSQL"} ·{" "}
                  {profile.host}:{profile.port} · {profile.database}
                  {profile.dbType === "postgres" ? ` · ${profile.schema}` : ""}
                </p>
                <p className="break-words">ユーザー: {profile.user}</p>
              </CardContent>
              <CardFooter className="flex flex-wrap justify-end gap-2 p-5 pt-4">
                {!isActive ? (
                  <Button
                    disabled={actionsDisabled}
                    size="sm"
                    type="button"
                    onClick={() => onActivate(profile.id)}
                  >
                    {operation === "activating"
                      ? "切替中…"
                      : "この接続先を使用"}
                  </Button>
                ) : null}
                <Button
                  disabled={actionsDisabled}
                  size="sm"
                  type="button"
                  variant="outline"
                  onClick={(event) => onEdit(profile, event.currentTarget)}
                >
                  編集
                </Button>
                <Button
                  disabled={actionsDisabled}
                  size="sm"
                  type="button"
                  variant="destructive"
                  onClick={(event) => onDelete(profile, event.currentTarget)}
                >
                  削除
                </Button>
              </CardFooter>
            </Card>
          </li>
        );
      })}
    </ul>
  );
}
