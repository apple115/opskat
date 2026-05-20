import { useTranslation } from "react-i18next";
import { Input, Label, Switch } from "@opskat/ui";
import { AssetSelect } from "@/components/asset/AssetSelect";
import { PasswordSourceField } from "@/components/asset/PasswordSourceField";
import { credential_entity } from "../../../wailsjs/go/models";

export interface EtcdConfigSectionProps {
  host: string;
  setHost: (v: string) => void;
  port: number;
  setPort: (v: number) => void;
  username: string;
  setUsername: (v: string) => void;
  tls: boolean;
  setTls: (v: boolean) => void;
  tlsInsecure: boolean;
  setTlsInsecure: (v: boolean) => void;
  tlsServerName: string;
  setTlsServerName: (v: string) => void;
  tlsCAFile: string;
  setTlsCAFile: (v: string) => void;
  tlsCertFile: string;
  setTlsCertFile: (v: string) => void;
  tlsKeyFile: string;
  setTlsKeyFile: (v: string) => void;
  commandTimeoutSeconds: number;
  setCommandTimeoutSeconds: (v: number) => void;
  sshTunnelId: number;
  setSshTunnelId: (v: number) => void;
  // Password fields
  password: string;
  setPassword: (v: string) => void;
  encryptedPassword: string;
  passwordSource: "inline" | "managed";
  setPasswordSource: (v: "inline" | "managed") => void;
  passwordCredentialId: number;
  setPasswordCredentialId: (v: number) => void;
  managedPasswords: credential_entity.Credential[];
  editAssetId?: number;
}

export function EtcdConfigSection({
  host,
  setHost,
  port,
  setPort,
  username,
  setUsername,
  tls,
  setTls,
  tlsInsecure,
  setTlsInsecure,
  tlsServerName,
  setTlsServerName,
  tlsCAFile,
  setTlsCAFile,
  tlsCertFile,
  setTlsCertFile,
  tlsKeyFile,
  setTlsKeyFile,
  commandTimeoutSeconds,
  setCommandTimeoutSeconds,
  sshTunnelId,
  setSshTunnelId,
  password,
  setPassword,
  passwordSource,
  setPasswordSource,
  passwordCredentialId,
  setPasswordCredentialId,
  managedPasswords,
  editAssetId,
}: EtcdConfigSectionProps) {
  const { t } = useTranslation();

  return (
    <div className="grid gap-4">
      <div className="grid grid-cols-2 gap-4">
        <div className="grid gap-2">
          <Label>{t("asset.host")}</Label>
          <Input value={host} onChange={(e) => setHost(e.target.value)} placeholder="127.0.0.1" />
        </div>
        <div className="grid gap-2">
          <Label>{t("asset.port")}</Label>
          <Input
            type="number"
            value={port || ""}
            onChange={(e) => setPort(Number(e.target.value))}
            placeholder="2379"
          />
        </div>
      </div>

      <div className="grid gap-2">
        <Label>{t("asset.username")}</Label>
        <Input value={username} onChange={(e) => setUsername(e.target.value)} placeholder={t("asset.usernameOptional")} />
      </div>

      <PasswordSourceField
        source={passwordSource}
        onSourceChange={setPasswordSource}
        password={password}
        onPasswordChange={setPassword}
        credentialId={passwordCredentialId}
        onCredentialIdChange={setPasswordCredentialId}
        managedPasswords={managedPasswords}
        editAssetId={editAssetId}
      />

      <div className="grid gap-2">
        <Label>{t("asset.sshTunnel")}</Label>
        <AssetSelect
          value={sshTunnelId}
          onValueChange={setSshTunnelId}
          filterType="ssh"
          placeholder={t("asset.selectSshTunnel")}
        />
      </div>

      <div className="flex items-center gap-2">
        <Switch checked={tls} onCheckedChange={setTls} id="etcd-tls" />
        <Label htmlFor="etcd-tls">{t("asset.useTLS")}</Label>
      </div>

      {tls && (
        <div className="grid gap-4 rounded-lg border p-4">
          <div className="flex items-center gap-2">
            <Switch checked={tlsInsecure} onCheckedChange={setTlsInsecure} id="etcd-tls-insecure" />
            <Label htmlFor="etcd-tls-insecure">{t("asset.tlsSkipVerify")}</Label>
          </div>
          <div className="grid gap-2">
            <Label>{t("asset.tlsServerName")}</Label>
            <Input value={tlsServerName} onChange={(e) => setTlsServerName(e.target.value)} />
          </div>
          <div className="grid gap-2">
            <Label>{t("asset.tlsCAFile")}</Label>
            <Input value={tlsCAFile} onChange={(e) => setTlsCAFile(e.target.value)} />
          </div>
          <div className="grid gap-2">
            <Label>{t("asset.tlsCertFile")}</Label>
            <Input value={tlsCertFile} onChange={(e) => setTlsCertFile(e.target.value)} />
          </div>
          <div className="grid gap-2">
            <Label>{t("asset.tlsKeyFile")}</Label>
            <Input value={tlsKeyFile} onChange={(e) => setTlsKeyFile(e.target.value)} />
          </div>
        </div>
      )}

      <div className="grid gap-2">
        <Label>{t("asset.commandTimeout")}</Label>
        <Input
          type="number"
          value={commandTimeoutSeconds || ""}
          onChange={(e) => setCommandTimeoutSeconds(Number(e.target.value))}
          placeholder="30"
        />
      </div>
    </div>
  );
}
