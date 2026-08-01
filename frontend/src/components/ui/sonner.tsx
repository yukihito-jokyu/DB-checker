import { useTheme } from "next-themes";
import type * as React from "react";
import { Toaster as Sonner } from "sonner";

type ToasterProps = React.ComponentProps<typeof Sonner>;

function Toaster(props: ToasterProps) {
  const { theme = "system" } = useTheme();

  return (
    <Sonner
      theme={theme as ToasterProps["theme"]}
      className="toaster group"
      toastOptions={{
        classNames: {
          toast:
            "group toast shadow-raised [&[data-type=default]]:border-border [&[data-type=default]]:bg-background [&[data-type=default]]:text-foreground",
          description: "group-[.toast]:text-muted-foreground",
          success:
            "!border-success-border !bg-success-muted !text-success [&_[data-icon]]:!text-success",
          error:
            "!border-destructive-border !bg-destructive-muted !text-destructive [&_[data-icon]]:!text-destructive",
          actionButton:
            "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
          cancelButton:
            "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground",
        },
      }}
      {...props}
    />
  );
}

export { Toaster };
