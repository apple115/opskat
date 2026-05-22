import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button, Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, Input, Textarea } from "@opskat/ui";
import { EtcdPutKey } from "../../../wailsjs/go/etcd/Etcd";

interface EtcdCreateKeyDialogProps {
  open: boolean;
  assetId: number;
  onOpenChange: (open: boolean) => void;
  onCreated: (key: string) => void | Promise<void>;
}

export function EtcdCreateKeyDialog({ open, assetId, onOpenChange, onCreated }: EtcdCreateKeyDialogProps) {
  const { t } = useTranslation();
  const [key, setKey] = useState("");
  const [value, setValue] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const reset = useCallback(() => {
    setKey("");
    setValue("");
    setSubmitting(false);
  }, []);

  const close = useCallback(() => {
    reset();
    onOpenChange(false);
  }, [onOpenChange, reset]);

  const submit = useCallback(async () => {
    const trimmedKey = key.trim();
    if (!trimmedKey) {
      toast.error(t("query.etcdKeyNameRequired"));
      return;
    }

    setSubmitting(true);
    try {
      await EtcdPutKey({ assetId, key: trimmedKey, value });
      await onCreated(trimmedKey);
      close();
    } catch (err) {
      toast.error(String(err));
      setSubmitting(false);
    }
  }, [assetId, key, value, onCreated, close, t]);

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (nextOpen) {
          onOpenChange(true);
        } else if (!submitting) {
          close();
        }
      }}
    >
      <DialogContent className="max-w-lg" showCloseButton={!submitting}>
        <DialogHeader>
          <DialogTitle>{t("query.createEtcdKey")}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t("query.etcdKeyName")}</label>
            <Input
              className="h-8 font-mono text-xs"
              placeholder={t("query.etcdKeyNamePlaceholder")}
              value={key}
              onChange={(e) => setKey(e.target.value)}
              disabled={submitting}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  submit();
                }
              }}
            />
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t("query.etcdKeyValue")}</label>
            <Textarea
              className="min-h-28 font-mono text-xs"
              placeholder={t("query.etcdKeyValuePlaceholder")}
              value={value}
              onChange={(e) => setValue(e.target.value)}
              disabled={submitting}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" size="sm" onClick={close} disabled={submitting}>
            {t("action.cancel")}
          </Button>
          <Button size="sm" onClick={submit} disabled={submitting}>
            {submitting ? <Loader2 className="mr-1 size-3 animate-spin" /> : null}
            {t("query.createEtcdKeySubmit")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
