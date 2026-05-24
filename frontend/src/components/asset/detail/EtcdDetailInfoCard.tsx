import { useTranslation } from "react-i18next";
import type { DetailInfoCardProps } from "@/lib/assetTypes/types";
import { DetailGrid, DetailSection, InfoItem, TunnelInfo } from "./InfoItem";
import { parseDetailConfig } from "./utils";

interface EtcdConfig {
  host: string;
  port: number;
  username?: string;
  tls?: boolean;
  ssh_asset_id?: number;
}

export function EtcdDetailInfoCard({ asset, sshTunnelName }: DetailInfoCardProps) {
  const { t } = useTranslation();

  const cfg = parseDetailConfig<EtcdConfig>(asset.Config);
  if (!cfg) return null;
  const tunnelName = sshTunnelName(cfg.ssh_asset_id);

  return (
    <DetailSection title={t("asset.connection")}>
      <DetailGrid>
        <InfoItem label={t("asset.host")} value={cfg.host} />
        <InfoItem label={t("asset.port")} value={String(cfg.port || 2379)} />
        <InfoItem label={t("asset.username")} value={cfg.username || "-"} />
        {cfg.tls && <InfoItem label={t("asset.tls")} value={"✓"} />}
      </DetailGrid>
      {tunnelName && <TunnelInfo label={t("asset.sshTunnel")} name={tunnelName} />}
    </DetailSection>
  );
}
