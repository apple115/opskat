import { useState, useEffect, useCallback, useRef } from "react";
import { useTranslation } from "react-i18next";
import { Button, Input, ScrollArea, Separator, Tabs, TabsContent, TabsList, TabsTrigger } from "@opskat/ui";
import {
  Search,
  RefreshCw,
  Trash2,
  Save,
  Users,
  Eye,
  EyeOff,
  Plus,
  List,
  FolderTree,
  ChevronRight,
  ChevronDown,
  Folder,
  FolderOpen,
  Key,
} from "lucide-react";
import { toast } from "sonner";
import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";
import { EtcdStartWatch, EtcdStopWatch } from "../../../wailsjs/go/etcd/Etcd";
import { etcd_svc } from "../../../wailsjs/go/models";
import { useQueryStore, type EtcdTabState } from "@/stores/queryStore";
import { EtcdCreateKeyDialog } from "./EtcdCreateKeyDialog";
import { buildKeyTree, flattenTree, type RedisFlatTreeRow } from "@/lib/redisKeyTree";

interface EtcdPanelProps {
  tabId: string;
}

const PAGE_SIZE = 100;

const FALLBACK_ETCD_STATE: EtcdTabState = {
  prefix: "",
  keys: [],
  loadingKeys: false,
  selectedKey: null,
  keyDetail: null,
  keyEditValue: "",
  status: null,
  members: [],
  activeTab: "keys",
  viewMode: "tree",
  treeExpanded: [],
  watchPrefix: "",
  isWatching: false,
  watchLogs: [],
  watchId: "",
  error: null,
};

export function EtcdPanel({ tabId }: EtcdPanelProps) {
  const { t } = useTranslation();
  const assetId = parseInt(tabId.replace("query-", ""), 10) || 0;

  const state = useQueryStore((s) => s.etcdStates[tabId]);
  const loadEtcdKeys = useQueryStore((s) => s.loadEtcdKeys);
  const selectEtcdKey = useQueryStore((s) => s.selectEtcdKey);
  const setEtcdPrefix = useQueryStore((s) => s.setEtcdPrefix);
  const setEtcdViewMode = useQueryStore((s) => s.setEtcdViewMode);
  const toggleEtcdTreeNode = useQueryStore((s) => s.toggleEtcdTreeNode);
  const setEtcdActiveTab = useQueryStore((s) => s.setEtcdActiveTab);
  const loadEtcdStatus = useQueryStore((s) => s.loadEtcdStatus);
  const loadEtcdMembers = useQueryStore((s) => s.loadEtcdMembers);
  const setEtcdWatchPrefix = useQueryStore((s) => s.setEtcdWatchPrefix);
  const setEtcdWatching = useQueryStore((s) => s.setEtcdWatching);
  const appendEtcdWatchLog = useQueryStore((s) => s.appendEtcdWatchLog);
  const clearEtcdWatchLogs = useQueryStore((s) => s.clearEtcdWatchLogs);
  const setEtcdKeyEditValue = useQueryStore((s) => s.setEtcdKeyEditValue);
  const putEtcdKey = useQueryStore((s) => s.putEtcdKey);
  const deleteEtcdKeys = useQueryStore((s) => s.deleteEtcdKeys);

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const logsEndRef = useRef<HTMLDivElement>(null);

  const {
    prefix,
    keys,
    loadingKeys: loading,
    selectedKey,
    keyDetail: detail,
    keyEditValue: editValue,
    status,
    members,
    activeTab,
    viewMode,
    treeExpanded,
    watchPrefix,
    isWatching,
    watchLogs,
    watchId,
  } = state ?? FALLBACK_ETCD_STATE;

  // Load keys on mount / prefix change
  useEffect(() => {
    if (assetId) {
      loadEtcdKeys(tabId);
    }
  }, [assetId, prefix, tabId, loadEtcdKeys]);

  // Load cluster info when tab switches to cluster
  useEffect(() => {
    if (activeTab === "cluster" && assetId) {
      loadEtcdStatus(tabId);
      loadEtcdMembers(tabId);
    }
  }, [activeTab, assetId, tabId, loadEtcdStatus, loadEtcdMembers]);

  // Watch event listener
  useEffect(() => {
    if (!isWatching || !assetId) return;
    const eventName = `etcd:watch:${assetId}`;
    const cancel = EventsOn(
      eventName,
      (event: {
        type: string;
        key: string;
        value?: string;
        prevValue?: string;
        revision: number;
        timestamp: number;
      }) => {
        appendEtcdWatchLog(tabId, {
          type: event.type,
          key: event.key,
          value: event.value,
          prevValue: event.prevValue,
          revision: event.revision,
          time: new Date(event.timestamp).toLocaleTimeString(),
        });
      }
    );
    return () => {
      cancel();
      EventsOff(eventName);
    };
  }, [isWatching, assetId, tabId, appendEtcdWatchLog]);

  // Auto-scroll watch logs
  useEffect(() => {
    if (logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [watchLogs]);

  // Cleanup watch on unmount
  useEffect(() => {
    return () => {
      if (watchId) {
        EtcdStopWatch(watchId);
        setEtcdWatching(tabId, false, "");
      }
    };
  }, [tabId, watchId, setEtcdWatching]);

  const startWatch = async () => {
    if (!assetId || isWatching) return;
    try {
      const watchID = await EtcdStartWatch(assetId, watchPrefix || "");
      setEtcdWatching(tabId, true, watchID);
      toast.success(t("query.watchStarted"));
    } catch (err) {
      toast.error(String(err));
    }
  };

  const stopWatch = () => {
    if (watchId) {
      EtcdStopWatch(watchId);
      setEtcdWatching(tabId, false, "");
    }
  };

  const handleSelectKey = (kv: etcd_svc.EtcdKeyValue) => {
    selectEtcdKey(tabId, kv.key);
  };

  const handleCreatedKey = async (createdKey: string) => {
    setEtcdPrefix(tabId, "");
    await loadEtcdKeys(tabId);
    await selectEtcdKey(tabId, createdKey);
    toast.success(t("query.createEtcdKeySuccess"));
  };

  const handlePut = async () => {
    if (!selectedKey) return;
    try {
      await putEtcdKey(tabId, selectedKey, editValue);
      toast.success(t("saveSuccess"));
      await loadEtcdKeys(tabId);
      await selectEtcdKey(tabId, selectedKey);
    } catch (err) {
      toast.error(String(err));
    }
  };

  const handleDelete = async (key: string) => {
    try {
      await deleteEtcdKeys(tabId, [key]);
      toast.success(t("deleteSuccess"));
      await loadEtcdKeys(tabId);
    } catch (err) {
      toast.error(String(err));
    }
  };

  const expandedSet = useCallback(() => new Set(treeExpanded), [treeExpanded]);

  return (
    <div className="flex h-full">
      {/* Left: Key list */}
      <div className="flex w-[320px] flex-col border-r">
        <div className="flex items-center gap-2 p-3">
          <Input
            value={prefix}
            onChange={(e) => setEtcdPrefix(tabId, e.target.value)}
            placeholder={t("query.etcdPrefix")}
            onKeyDown={(e) => e.key === "Enter" && loadEtcdKeys(tabId)}
          />
          <Button size="icon" variant="ghost" onClick={() => loadEtcdKeys(tabId)} disabled={loading}>
            <Search className="h-4 w-4" />
          </Button>
          <Button size="icon" variant="ghost" onClick={() => loadEtcdKeys(tabId)} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
          <Button
            size="icon"
            variant="ghost"
            onClick={() => setEtcdViewMode(tabId, viewMode === "list" ? "tree" : "list")}
            title={viewMode === "list" ? t("query.treeView") : t("query.listView")}
          >
            {viewMode === "list" ? <FolderTree className="h-4 w-4" /> : <List className="h-4 w-4" />}
          </Button>
          <Button
            size="icon"
            variant="ghost"
            onClick={() => setCreateDialogOpen(true)}
            title={t("query.createEtcdKey")}
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>
        <Separator />
        <ScrollArea className="flex-1">
          <div className="p-2 space-y-1">
            {viewMode === "tree" && keys.length > 0
              ? (() => {
                  const tree = buildKeyTree(
                    keys.map((k) => k.key),
                    "/"
                  );
                  const rows = flattenTree(tree, expandedSet(), "/");
                  return rows.map((row: RedisFlatTreeRow) => {
                    const isKey = row.fullKey !== null;
                    const isFolder = row.hasChildren;
                    return (
                      <div
                        key={row.nodeId}
                        className={`flex cursor-pointer items-center justify-between rounded px-2 py-1.5 text-sm ${
                          isKey && selectedKey === row.fullKey ? "bg-accent text-accent-foreground" : "hover:bg-muted"
                        }`}
                        style={{ paddingLeft: `${row.depth * 16 + 8}px` }}
                        onClick={() => {
                          if (isKey) {
                            const kv = keys.find((k) => k.key === row.fullKey);
                            if (kv) handleSelectKey(kv);
                          } else if (isFolder) {
                            toggleEtcdTreeNode(tabId, row.nodeId);
                          }
                        }}
                      >
                        <div className="flex items-center gap-1.5 min-w-0 flex-1">
                          {isFolder ? (
                            <button
                              type="button"
                              className="flex size-4 shrink-0 items-center justify-center rounded-sm hover:bg-accent"
                              onClick={(e) => {
                                e.stopPropagation();
                                toggleEtcdTreeNode(tabId, row.nodeId);
                              }}
                            >
                              {row.isExpanded ? (
                                <ChevronDown className="size-3 text-muted-foreground" />
                              ) : (
                                <ChevronRight className="size-3 text-muted-foreground" />
                              )}
                            </button>
                          ) : (
                            <span className="size-4 shrink-0" />
                          )}
                          {isKey ? (
                            <Key className="size-3 shrink-0 text-muted-foreground" />
                          ) : row.isExpanded ? (
                            <FolderOpen className="size-3 shrink-0 text-muted-foreground" />
                          ) : (
                            <Folder className="size-3 shrink-0 text-muted-foreground" />
                          )}
                          <span className="truncate font-mono">{row.name}</span>
                          {isFolder && (
                            <span className="ml-auto shrink-0 text-muted-foreground text-[10px]">{row.keyCount}</span>
                          )}
                        </div>
                        {isKey && (
                          <Button
                            size="icon"
                            variant="ghost"
                            className="h-6 w-6 opacity-0 group-hover:opacity-100 shrink-0"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDelete(row.fullKey!);
                            }}
                          >
                            <Trash2 className="h-3 w-3 text-destructive" />
                          </Button>
                        )}
                      </div>
                    );
                  });
                })()
              : keys.map((kv) => (
                  <div
                    key={kv.key}
                    className={`flex cursor-pointer items-center justify-between rounded px-2 py-1.5 text-sm group ${
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
        <Tabs
          value={activeTab}
          onValueChange={(v) => setEtcdActiveTab(tabId, v as "keys" | "cluster" | "watch")}
          className="flex flex-1 flex-col"
        >
          <div className="border-b px-4">
            <TabsList>
              <TabsTrigger value="keys">{t("query.keys")}</TabsTrigger>
              <TabsTrigger value="watch">{t("query.watch")}</TabsTrigger>
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
                      {t("save")}
                    </Button>
                    <Button size="sm" variant="destructive" onClick={() => handleDelete(detail.key)}>
                      <Trash2 className="mr-1 h-4 w-4" />
                      {t("delete")}
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
                  onChange={(e) => setEtcdKeyEditValue(tabId, e.target.value)}
                />
              </div>
            ) : (
              <div className="flex flex-1 items-center justify-center text-muted-foreground">
                {t("query.selectKey")}
              </div>
            )}
          </TabsContent>

          <TabsContent value="watch" className="flex flex-1 flex-col m-0 p-4 gap-4">
            <div className="flex items-center gap-2">
              <Input
                value={watchPrefix}
                onChange={(e) => setEtcdWatchPrefix(tabId, e.target.value)}
                placeholder={t("query.watchPrefix")}
                disabled={isWatching}
                className="flex-1"
              />
              {!isWatching ? (
                <Button onClick={startWatch}>
                  <Eye className="mr-1 h-4 w-4" />
                  {t("query.startWatch")}
                </Button>
              ) : (
                <Button variant="destructive" onClick={stopWatch}>
                  <EyeOff className="mr-1 h-4 w-4" />
                  {t("query.stopWatch")}
                </Button>
              )}
              <Button variant="outline" onClick={() => clearEtcdWatchLogs(tabId)} disabled={watchLogs.length === 0}>
                {t("query.clearWatchLogs")}
              </Button>
            </div>
            <div className="flex-1 overflow-auto rounded-md border">
              <table className="w-full text-sm">
                <thead className="bg-muted sticky top-0">
                  <tr>
                    <th className="px-3 py-2 text-left font-medium">{t("query.watchTime")}</th>
                    <th className="px-3 py-2 text-left font-medium">{t("query.watchType")}</th>
                    <th className="px-3 py-2 text-left font-medium">Key</th>
                    <th className="px-3 py-2 text-left font-medium">Value</th>
                    <th className="px-3 py-2 text-left font-medium">Revision</th>
                  </tr>
                </thead>
                <tbody>
                  {watchLogs.map((log, idx) => (
                    <tr key={idx} className="border-t">
                      <td className="px-3 py-2 font-mono text-xs text-muted-foreground">{log.time}</td>
                      <td className="px-3 py-2">
                        <span
                          className={`inline-flex rounded px-1.5 py-0.5 text-xs font-medium ${
                            log.type === "put"
                              ? "bg-green-100 text-green-800"
                              : log.type === "delete"
                                ? "bg-red-100 text-red-800"
                                : "bg-gray-100 text-gray-800"
                          }`}
                        >
                          {log.type === "put"
                            ? t("query.watchEventPut")
                            : log.type === "delete"
                              ? t("query.watchEventDelete")
                              : log.type}
                        </span>
                      </td>
                      <td className="px-3 py-2 font-mono text-xs truncate max-w-[200px]">{log.key}</td>
                      <td className="px-3 py-2 font-mono text-xs truncate max-w-[200px]">{log.value || "-"}</td>
                      <td className="px-3 py-2 font-mono text-xs text-muted-foreground">{log.revision}</td>
                    </tr>
                  ))}
                  {watchLogs.length === 0 && (
                    <tr>
                      <td colSpan={5} className="px-3 py-8 text-center text-sm text-muted-foreground">
                        {isWatching ? t("query.watchWaiting") : t("query.noWatchLogs")}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
              <div ref={logsEndRef} />
            </div>
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

      <EtcdCreateKeyDialog
        open={createDialogOpen}
        assetId={assetId}
        onOpenChange={setCreateDialogOpen}
        onCreated={handleCreatedKey}
      />
    </div>
  );
}
