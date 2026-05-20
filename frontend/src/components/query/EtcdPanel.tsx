import { useState, useEffect, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Button, Input, ScrollArea, Separator, Tabs, TabsContent, TabsList, TabsTrigger } from "@opskat/ui";
import { Search, RefreshCw, Trash2, Save, Users } from "lucide-react";
import { toast } from "sonner";
import {
  EtcdRangeKeys,
  EtcdGetKey,
  EtcdPutKey,
  EtcdDeleteKeys,
  EtcdGetStatus,
  EtcdGetMembers,
} from "../../../wailsjs/go/etcd/Etcd";

interface EtcdPanelProps {
  tabId: string;
}

interface EtcdKV {
  key: string;
  value: string;
  createRevision: number;
  modRevision: number;
  version: number;
  lease: number;
}

interface EtcdMember {
  id: number;
  name: string;
  peerURLs: string[];
  clientURLs: string[];
  isLearner: boolean;
}

interface EtcdStatusData {
  version: string;
  dbSize: number;
  leader: number;
  raftIndex: number;
  raftTerm: number;
}

const PAGE_SIZE = 100;

export function EtcdPanel({ tabId }: EtcdPanelProps) {
  const { t } = useTranslation();
  const assetId = parseInt(tabId.replace("query-", ""), 10) || 0;

  const [prefix, setPrefix] = useState("");
  const [keys, setKeys] = useState<EtcdKV[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [detail, setDetail] = useState<EtcdKV | null>(null);
  const [editValue, setEditValue] = useState("");
  const [status, setStatus] = useState<EtcdStatusData | null>(null);
  const [members, setMembers] = useState<EtcdMember[]>([]);
  const [activeTab, setActiveTab] = useState("keys");

  const fetchKeys = useCallback(async () => {
    if (!assetId) return;
    setLoading(true);
    try {
      const result = await EtcdRangeKeys({
        assetId,
        prefix: prefix || "",
        limit: PAGE_SIZE,
      });
      setKeys(result.keys || []);
    } catch (err) {
      toast.error(String(err));
    } finally {
      setLoading(false);
    }
  }, [assetId, prefix]);

  const fetchDetail = useCallback(
    async (key: string) => {
      if (!assetId) return;
      try {
        const result = await EtcdGetKey({ assetId, key });
        setDetail(result);
        setEditValue(result.value || "");
      } catch (err) {
        toast.error(String(err));
      }
    },
    [assetId]
  );

  const fetchStatus = useCallback(async () => {
    if (!assetId) return;
    try {
      const result = await EtcdGetStatus(assetId);
      setStatus(result);
    } catch (err) {
      toast.error(String(err));
    }
  }, [assetId]);

  const fetchMembers = useCallback(async () => {
    if (!assetId) return;
    try {
      const result = await EtcdGetMembers(assetId);
      setMembers(result || []);
    } catch (err) {
      toast.error(String(err));
    }
  }, [assetId]);

  useEffect(() => {
    fetchKeys();
  }, [fetchKeys]);

  useEffect(() => {
    if (activeTab === "cluster") {
      fetchStatus();
      fetchMembers();
    }
  }, [activeTab, fetchStatus, fetchMembers]);

  const handleSelectKey = (kv: EtcdKV) => {
    setSelectedKey(kv.key);
    fetchDetail(kv.key);
  };

  const handlePut = async () => {
    if (!assetId || !selectedKey) return;
    try {
      await EtcdPutKey({ assetId, key: selectedKey, value: editValue });
      toast.success(t("saveSuccess"));
      fetchKeys();
      fetchDetail(selectedKey);
    } catch (err) {
      toast.error(String(err));
    }
  };

  const handleDelete = async (key: string) => {
    if (!assetId) return;
    try {
      await EtcdDeleteKeys({ assetId, keys: [key] });
      toast.success(t("deleteSuccess"));
      setSelectedKey(null);
      setDetail(null);
      fetchKeys();
    } catch (err) {
      toast.error(String(err));
    }
  };

  return (
    <div className="flex h-full">
      {/* Left: Key list */}
      <div className="flex w-[320px] flex-col border-r">
        <div className="flex items-center gap-2 p-3">
          <Input
            value={prefix}
            onChange={(e) => setPrefix(e.target.value)}
            placeholder={t("query.etcdPrefix")}
            onKeyDown={(e) => e.key === "Enter" && fetchKeys()}
          />
          <Button size="icon" variant="ghost" onClick={fetchKeys} disabled={loading}>
            <Search className="h-4 w-4" />
          </Button>
          <Button size="icon" variant="ghost" onClick={fetchKeys} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
        </div>
        <Separator />
        <ScrollArea className="flex-1">
          <div className="p-2 space-y-1">
            {keys.map((kv) => (
              <div
                key={kv.key}
                className={`flex cursor-pointer items-center justify-between rounded px-2 py-1.5 text-sm ${
                  selectedKey === kv.key ? "bg-accent" : "hover:bg-muted"
                }`}
                onClick={() => handleSelectKey(kv)}
              >
                <span className="truncate flex-1 font-mono">{kv.key}</span>
                <Button
                  size="icon"
                  variant="ghost"
                  className="h-6 w-6 opacity-0 group-hover:opacity-100"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDelete(kv.key);
                  }}
                >
                  <Trash2 className="h-3 w-3 text-destructive" />
                </Button>
              </div>
            ))}
            {keys.length === 0 && !loading && (
              <div className="p-4 text-center text-sm text-muted-foreground">{t("query.noKeys")}</div>
            )}
          </div>
        </ScrollArea>
        {keys.length >= PAGE_SIZE && (
          <div className="p-2 text-center text-xs text-muted-foreground">
            {t("query.showingNKeys", { count: keys.length })}
          </div>
        )}
      </div>

      {/* Right: Detail / Cluster */}
      <div className="flex flex-1 flex-col">
        <Tabs value={activeTab} onValueChange={setActiveTab} className="flex flex-1 flex-col">
          <div className="border-b px-4">
            <TabsList>
              <TabsTrigger value="keys">{t("query.keys")}</TabsTrigger>
              <TabsTrigger value="cluster">{t("query.cluster")}</TabsTrigger>
            </TabsList>
          </div>

          <TabsContent value="keys" className="flex flex-1 flex-col m-0">
            {selectedKey && detail ? (
              <div className="flex flex-1 flex-col p-4 gap-4">
                <div className="flex items-center justify-between">
                  <div className="font-mono text-sm font-medium">{detail.key}</div>
                  <div className="flex gap-2">
                    <Button size="sm" onClick={handlePut}>
                      <Save className="mr-1 h-4 w-4" />
                      {t("common.save")}
                    </Button>
                    <Button size="sm" variant="destructive" onClick={() => handleDelete(detail.key)}>
                      <Trash2 className="mr-1 h-4 w-4" />
                      {t("common.delete")}
                    </Button>
                  </div>
                </div>

                <div className="grid grid-cols-4 gap-2 text-xs text-muted-foreground">
                  <div>Version: {detail.version}</div>
                  <div>CreateRev: {detail.createRevision}</div>
                  <div>ModRev: {detail.modRevision}</div>
                  <div>Lease: {detail.lease || "-"}</div>
                </div>

                <textarea
                  className="flex-1 rounded-md border bg-background p-3 font-mono text-sm resize-none"
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                />
              </div>
            ) : (
              <div className="flex flex-1 items-center justify-center text-muted-foreground">
                {t("query.selectKey")}
              </div>
            )}
          </TabsContent>

          <TabsContent value="cluster" className="flex flex-1 flex-col m-0 p-4 gap-4">
            {status && (
              <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
                <div className="rounded-lg border p-3">
                  <div className="text-xs text-muted-foreground">Version</div>
                  <div className="text-lg font-semibold">{status.version}</div>
                </div>
                <div className="rounded-lg border p-3">
                  <div className="text-xs text-muted-foreground">DB Size</div>
                  <div className="text-lg font-semibold">{(status.dbSize / 1024 / 1024).toFixed(2)} MB</div>
                </div>
                <div className="rounded-lg border p-3">
                  <div className="text-xs text-muted-foreground">Raft Index</div>
                  <div className="text-lg font-semibold">{status.raftIndex}</div>
                </div>
                <div className="rounded-lg border p-3">
                  <div className="text-xs text-muted-foreground">Raft Term</div>
                  <div className="text-lg font-semibold">{status.raftTerm}</div>
                </div>
              </div>
            )}

            <div className="rounded-lg border">
              <div className="border-b px-4 py-2 text-sm font-medium">{t("query.members")}</div>
              <div className="divide-y">
                {members.map((m) => (
                  <div key={m.id} className="flex items-center justify-between px-4 py-2 text-sm">
                    <div className="flex items-center gap-2">
                      <Users className="h-4 w-4 text-muted-foreground" />
                      <span className="font-medium">{m.name || `Member ${m.id}`}</span>
                      {m.isLearner && (
                        <span className="rounded bg-yellow-100 px-1.5 py-0.5 text-xs text-yellow-800">Learner</span>
                      )}
                    </div>
                    <div className="text-muted-foreground font-mono text-xs">{m.clientURLs.join(", ")}</div>
                  </div>
                ))}
                {members.length === 0 && (
                  <div className="px-4 py-4 text-center text-sm text-muted-foreground">{t("query.noMembers")}</div>
                )}
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
