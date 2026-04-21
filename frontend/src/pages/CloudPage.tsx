import { FormEvent, useEffect, useMemo, useState } from "react";

import type { PageData } from "@/api/types";
import {
  createCloudAccount,
  createCloudAsset,
  deleteCloudAccount,
  deleteCloudAsset,
  listCloudAccountAssets,
  listCloudAccounts,
  listCloudAssets,
  syncCloudAccount,
  updateCloudAccount,
  updateCloudAsset,
  verifyCloudAccount,
} from "@/api/cloud";
import { DeleteConfirmModal } from "@/components/DeleteConfirmModal";
import type { TableSettingsColumn } from "@/components/TableSettingsModal";
import { TableSettingsModal } from "@/components/TableSettingsModal";
import { PermissionButton } from "@/components/PermissionButton";
import { RowActionOverflow } from "@/components/RowActionOverflow";
import type { CloudAccountItem, CloudAssetItem, CloudAssetType, CloudProvider } from "@/types/cloud";
import {
  loadPersistedListSettings,
  sanitizeVisibleColumnKeys,
  savePersistedListSettings,
} from "@/utils/listSettings";
import { showToast } from "@/utils/toast";

const defaultPageSize = 10;
const pageSizeOptions = [10, 20, 50];
const providerOptions: CloudProvider[] = ["aws", "aliyun", "tencent", "huawei"];
const CLOUD_ACCOUNT_LIST_SETTINGS_KEY = "cloud.accounts.table.settings";
const CLOUD_ASSET_LIST_SETTINGS_KEY = "cloud.assets.table.settings";
const defaultAccountVisibleColumnKeys = ["id", "provider", "name", "region", "verified", "updatedAt", "actions"];
const defaultAssetVisibleColumnKeys = ["id", "providerAccount", "type", "resourceId", "name", "status", "source", "expiresAt", "lastSyncedAt", "actions"];
const cloudAssetTypeOptions: CloudAssetType[] = [
  "CloudServer",
  "MySQL",
  "PrivateNetwork",
  "ObjectStorage",
  "FileStorage",
  "ContainerService",
  "LoadBalancer",
  "DNS",
  "SSLCertificate",
  "LogService",
];
const cloudAccountTableColumns: TableSettingsColumn[] = [
  { key: "id", label: "ID" },
  { key: "provider", label: "云厂商" },
  { key: "name", label: "名称" },
  { key: "region", label: "地域" },
  { key: "verified", label: "校验状态" },
  { key: "updatedAt", label: "更新时间" },
  { key: "actions", label: "操作", required: true },
];
const cloudAssetTableColumns: TableSettingsColumn[] = [
  { key: "id", label: "ID" },
  { key: "providerAccount", label: "provider/account" },
  { key: "type", label: "类型" },
  { key: "resourceId", label: "resourceId" },
  { key: "name", label: "名称" },
  { key: "status", label: "状态" },
  { key: "source", label: "来源" },
  { key: "expiresAt", label: "过期时间" },
  { key: "lastSyncedAt", label: "最近同步" },
  { key: "actions", label: "操作", required: true },
];

interface AssetTemplatePreset {
  label: string;
  tags: Record<string, unknown>;
  metadata: Record<string, unknown>;
}

type TableSettingsTarget = "closed" | "accounts" | "assets";
type KnownCloudAssetType = typeof cloudAssetTypeOptions[number];
type AssetTemplateKey = KnownCloudAssetType | "custom";

const cloudAssetTemplateMap: Record<KnownCloudAssetType, AssetTemplatePreset> = {
  CloudServer: {
    label: "云服务器模板",
    tags: { env: "prod", app: "web", tier: "frontend" },
    metadata: { cpu: 4, memoryGiB: 16, os: "linux", zone: "ap-southeast-1a" },
  },
  MySQL: {
    label: "云数据库 MySQL 模板",
    tags: { env: "prod", app: "order", tier: "database" },
    metadata: { engine: "MySQL", version: "8.0", storageGiB: 200, highAvailability: true },
  },
  PrivateNetwork: {
    label: "私有网络模板",
    tags: { env: "prod", network: "core" },
    metadata: { cidr: "10.10.0.0/16", subnetCount: 3, routeTable: "rtb-core" },
  },
  ObjectStorage: {
    label: "对象存储模板",
    tags: { env: "prod", dataClass: "archive" },
    metadata: { storageClass: "standard", versioning: true, encryption: "AES256" },
  },
  FileStorage: {
    label: "文件存储模板",
    tags: { env: "prod", workload: "shared-files" },
    metadata: { protocol: "NFS", throughputMBps: 200, capacityGiB: 1024 },
  },
  ContainerService: {
    label: "容器服务模板",
    tags: { env: "prod", platform: "kubernetes" },
    metadata: { clusterVersion: "1.29", nodeCount: 6, networkPlugin: "cni" },
  },
  LoadBalancer: {
    label: "负载均衡模板",
    tags: { env: "prod", app: "gateway" },
    metadata: { scheme: "public", listener: "443", healthCheckPath: "/healthz" },
  },
  DNS: {
    label: "域名管理模板",
    tags: { env: "prod", service: "dns" },
    metadata: { zone: "example.com", recordType: "A", ttl: 600 },
  },
  SSLCertificate: {
    label: "SSL证书模板",
    tags: { env: "prod", security: "tls" },
    metadata: { issuer: "LetsEncrypt", domain: "api.example.com", autoRenew: true },
  },
  LogService: {
    label: "日志服务模板",
    tags: { env: "prod", observability: "logs" },
    metadata: { retentionDays: 30, index: "app-*", format: "json" },
  },
};

type DrawerState =
  | { type: "closed" }
  | { type: "account-create" }
  | { type: "account-edit" }
  | { type: "asset-create" }
  | { type: "asset-edit" };

interface CloudAccountFormState {
  provider: CloudProvider;
  name: string;
  accessKey: string;
  secretKey: string;
  region: string;
}

interface CloudAssetFilterState {
  provider: string;
  accountId: string;
  region: string;
  type: string;
  keyword: string;
}

interface CloudAccountFilterState {
  provider: string;
  region: string;
  verified: string;
  keyword: string;
}

interface CloudAssetFormState {
  provider: CloudProvider;
  accountId: string;
  region: string;
  type: CloudAssetType;
  resourceId: string;
  name: string;
  status: string;
  source: string;
  expiresAt: string;
  tagsJSON: string;
  metadataJSON: string;
}

function defaultCloudAccountForm(): CloudAccountFormState {
  return {
    provider: "aws",
    name: "",
    accessKey: "",
    secretKey: "",
    region: "ap-southeast-1",
  };
}

function defaultCloudAssetFilter(): CloudAssetFilterState {
  return {
    provider: "",
    accountId: "",
    region: "",
    type: "",
    keyword: "",
  };
}

function defaultCloudAccountFilter(): CloudAccountFilterState {
  return {
    provider: "",
    region: "",
    verified: "",
    keyword: "",
  };
}

function defaultCloudAssetForm(): CloudAssetFormState {
  return {
    provider: "aws",
    accountId: "",
    region: "ap-southeast-1",
    type: "CloudServer",
    resourceId: "",
    name: "",
    status: "active",
    source: "Manual",
    expiresAt: "",
    tagsJSON: "{}",
    metadataJSON: "{}",
  };
}

export function CloudPage() {
  const [accounts, setAccounts] = useState<CloudAccountItem[]>([]);
  const [accountTotal, setAccountTotal] = useState(0);
  const [accountPage, setAccountPage] = useState(1);
  const [accountPageSize, setAccountPageSize] = useState(defaultPageSize);
  const [accountLoading, setAccountLoading] = useState(false);
  const [accountSubmitting, setAccountSubmitting] = useState(false);
  const [accountVerifyID, setAccountVerifyID] = useState<number | null>(null);
  const [accountSyncID, setAccountSyncID] = useState<number | null>(null);
  const [accountDeleteID, setAccountDeleteID] = useState<number | null>(null);
  const [deleteAccountTarget, setDeleteAccountTarget] = useState<CloudAccountItem | null>(null);
  const [accountEditID, setAccountEditID] = useState<number | null>(null);
  const [accountForm, setAccountForm] = useState<CloudAccountFormState>(defaultCloudAccountForm);
  const [accountFilter, setAccountFilter] = useState<CloudAccountFilterState>(defaultCloudAccountFilter);
  const [accountQuery, setAccountQuery] = useState<CloudAccountFilterState>(defaultCloudAccountFilter);

  const [assets, setAssets] = useState<CloudAssetItem[]>([]);
  const [assetTotal, setAssetTotal] = useState(0);
  const [assetPage, setAssetPage] = useState(1);
  const [assetPageSize, setAssetPageSize] = useState(defaultPageSize);
  const [assetLoading, setAssetLoading] = useState(false);
  const [assetSubmitting, setAssetSubmitting] = useState(false);
  const [assetDeleteID, setAssetDeleteID] = useState<number | null>(null);
  const [deleteAssetTarget, setDeleteAssetTarget] = useState<CloudAssetItem | null>(null);
  const [assetEditID, setAssetEditID] = useState<number | null>(null);
  const [assetFilter, setAssetFilter] = useState<CloudAssetFilterState>(defaultCloudAssetFilter);
  const [assetQuery, setAssetQuery] = useState<CloudAssetFilterState>(defaultCloudAssetFilter);
  const [assetForm, setAssetForm] = useState<CloudAssetFormState>(defaultCloudAssetForm);
  const [selectedAccountID, setSelectedAccountID] = useState<number | null>(null);
  const [drawer, setDrawer] = useState<DrawerState>({ type: "closed" });
  const [tableSettingsTarget, setTableSettingsTarget] = useState<TableSettingsTarget>("closed");
  const [selectedAssetTemplate, setSelectedAssetTemplate] = useState<AssetTemplateKey>("custom");
  const [visibleAccountColumnKeys, setVisibleAccountColumnKeys] = useState<string[]>(() => {
    const persisted = loadPersistedListSettings(CLOUD_ACCOUNT_LIST_SETTINGS_KEY);
    const defaults = sanitizeVisibleColumnKeys(defaultAccountVisibleColumnKeys, cloudAccountTableColumns);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaults, cloudAccountTableColumns);
  });
  const [visibleAssetColumnKeys, setVisibleAssetColumnKeys] = useState<string[]>(() => {
    const persisted = loadPersistedListSettings(CLOUD_ASSET_LIST_SETTINGS_KEY);
    const defaults = sanitizeVisibleColumnKeys(defaultAssetVisibleColumnKeys, cloudAssetTableColumns);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaults, cloudAssetTableColumns);
  });

  useEffect(() => {
    void loadAccountPage(accountPage, accountPageSize, accountQuery);
  }, [accountPage, accountPageSize, accountQuery]);

  useEffect(() => {
    void loadAssetPage(assetPage, assetPageSize, assetQuery, selectedAccountID);
  }, [assetPage, assetPageSize, assetQuery, selectedAccountID]);

  useEffect(() => {
    savePersistedListSettings(CLOUD_ACCOUNT_LIST_SETTINGS_KEY, {
      visibleColumnKeys: visibleAccountColumnKeys,
    });
  }, [visibleAccountColumnKeys]);

  useEffect(() => {
    savePersistedListSettings(CLOUD_ASSET_LIST_SETTINGS_KEY, {
      visibleColumnKeys: visibleAssetColumnKeys,
    });
  }, [visibleAssetColumnKeys]);

  const accountTotalPages = useMemo(() => totalPages(accountTotal, accountPageSize), [accountTotal, accountPageSize]);
  const assetTotalPages = useMemo(() => totalPages(assetTotal, assetPageSize), [assetTotal, assetPageSize]);
  const accountVisibleColumnSet = useMemo(() => new Set(visibleAccountColumnKeys), [visibleAccountColumnKeys]);
  const assetVisibleColumnSet = useMemo(() => new Set(visibleAssetColumnKeys), [visibleAssetColumnKeys]);
  const accountColSpan = Math.max(1, visibleAccountColumnKeys.length);
  const assetColSpan = Math.max(1, visibleAssetColumnKeys.length);

  async function loadAccountPage(page: number, pageSize: number, filters: CloudAccountFilterState) {
    setAccountLoading(true);
    try {
      const data = await listCloudAccounts({
        page,
        pageSize,
        keyword: filters.keyword || undefined,
        provider: filters.provider || undefined,
        region: filters.region || undefined,
        verified: filters.verified || undefined,
      });
      setAccounts(data.list);
      setAccountTotal(data.total);
    } catch {
      showToast("云账号加载失败");
    } finally {
      setAccountLoading(false);
    }
  }

  async function loadAssetPage(page: number, pageSize: number, filters: CloudAssetFilterState, accountID: number | null) {
    setAssetLoading(true);
    try {
      let data: PageData<CloudAssetItem>;
      if (accountID) {
        data = await listCloudAccountAssets(accountID, {
          page,
          pageSize,
          region: filters.region || undefined,
          type: filters.type || undefined,
        });
      } else {
        data = await listCloudAssets({
          page,
          pageSize,
          provider: filters.provider || undefined,
          accountId: parseOptionalNumber(filters.accountId),
          region: filters.region || undefined,
          type: filters.type || undefined,
          keyword: filters.keyword || undefined,
        });
      }
      setAssets(data.list);
      setAssetTotal(data.total);
    } catch {
      showToast("云资源加载失败");
    } finally {
      setAssetLoading(false);
    }
  }

  function closeDrawer() {
    setDrawer({ type: "closed" });
    setAccountEditID(null);
    setAssetEditID(null);
    setAccountForm(defaultCloudAccountForm());
    setAssetForm(defaultCloudAssetForm());
    setSelectedAssetTemplate("custom");
  }

  function openAccountCreateDrawer() {
    setAccountEditID(null);
    setAccountForm(defaultCloudAccountForm());
    setDrawer({ type: "account-create" });
  }

  function openAccountEditDrawer(item: CloudAccountItem) {
    setAccountEditID(item.id);
    setAccountForm({
      provider: item.provider,
      name: item.name,
      accessKey: "",
      secretKey: "",
      region: item.region ?? "",
    });
    setDrawer({ type: "account-edit" });
  }

  function openAssetCreateDrawer() {
    const form = defaultCloudAssetForm();
    if (selectedAccountID) {
      form.accountId = String(selectedAccountID);
    }
    const provider = normalizeProvider(assetQuery.provider);
    if (provider) {
      form.provider = provider;
    }
    if (assetQuery.region) {
      form.region = assetQuery.region;
    }
    const type = normalizeCloudAssetType(assetQuery.type);
    if (type) {
      form.type = type;
    }
    setAssetEditID(null);
    setAssetForm(form);
    setSelectedAssetTemplate(normalizeCloudAssetType(form.type) ?? "custom");
    setDrawer({ type: "asset-create" });
  }

  function openAssetEditDrawer(item: CloudAssetItem) {
    setAssetEditID(item.id);
    setAssetForm({
      provider: item.provider,
      accountId: item.accountId ? String(item.accountId) : "",
      region: item.region ?? "",
      type: item.type,
      resourceId: item.resourceId,
      name: item.name,
      status: item.status ?? "active",
      source: item.source ?? "Manual",
      expiresAt: item.expiresAt ?? "",
      tagsJSON: JSON.stringify(item.tags ?? {}, null, 2),
      metadataJSON: JSON.stringify(item.metadata ?? {}, null, 2),
    });
    setSelectedAssetTemplate("custom");
    setDrawer({ type: "asset-edit" });
  }

  function toggleAccountVisibleColumn(columnKey: string) {
    setVisibleAccountColumnKeys((prev) => {
      const exists = prev.includes(columnKey);
      if (exists) return prev.filter((key) => key !== columnKey);
      return [...prev, columnKey];
    });
  }

  function toggleAssetVisibleColumn(columnKey: string) {
    setVisibleAssetColumnKeys((prev) => {
      const exists = prev.includes(columnKey);
      if (exists) return prev.filter((key) => key !== columnKey);
      return [...prev, columnKey];
    });
  }

  function handleApplyAssetTemplate() {
    if (selectedAssetTemplate === "custom") return;
    const preset = cloudAssetTemplateMap[selectedAssetTemplate];
    setAssetForm((prev) => ({
      ...prev,
      type: selectedAssetTemplate,
      tagsJSON: JSON.stringify(preset.tags, null, 2),
      metadataJSON: JSON.stringify(preset.metadata, null, 2),
    }));
    showToast(`已应用 ${preset.label}`);
  }

  async function handleSubmitAccount(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!accountForm.name.trim()) {
      showToast("请填写账号名称");
      return;
    }
    const accessKey = accountForm.accessKey.trim();
    const secretKey = accountForm.secretKey.trim();
    if (!accountEditID && (!accessKey || !secretKey)) {
      showToast("请填写完整云账号信息");
      return;
    }
    if (accessKey.includes("*") || secretKey.includes("*")) {
      showToast("请填写原始密钥，不可使用脱敏值");
      return;
    }
    if (accountForm.provider === "tencent" && accessKey && !isLikelyTencentSecretId(accessKey)) {
      showToast("腾讯云 accessKey 应填写 SecretId（通常以 AKID 开头）");
      return;
    }
    setAccountSubmitting(true);
    try {
      const payload: Record<string, string> = {
        provider: accountForm.provider,
        name: accountForm.name.trim(),
        region: accountForm.region.trim(),
      };
      if (accessKey) payload.accessKey = accessKey;
      if (secretKey) payload.secretKey = secretKey;
      if (accountEditID) {
        await updateCloudAccount(accountEditID, payload);
      } else {
        await createCloudAccount(payload);
      }
      closeDrawer();
      await loadAccountPage(accountPage, accountPageSize, accountQuery);
      showToast("云账号保存成功");
    } catch (error) {
      showToast(extractErrorMessage(error, "云账号保存失败"));
    } finally {
      setAccountSubmitting(false);
    }
  }

  async function handleVerifyAccount(accountID: number) {
    setAccountVerifyID(accountID);
    try {
      await verifyCloudAccount(accountID);
      await loadAccountPage(accountPage, accountPageSize, accountQuery);
      showToast("云账号校验成功");
    } catch (error) {
      showToast(extractErrorMessage(error, "云账号校验失败"));
    } finally {
      setAccountVerifyID(null);
    }
  }

  async function handleSyncAccount(accountID: number) {
    setAccountSyncID(accountID);
    try {
      const result = await syncCloudAccount(accountID);
      if (result.job?.status === "failed") {
        showToast("同步任务执行失败");
      } else {
        const cloudCount = result.cloudAssetCount ?? 0;
        const cmdbCount = result.cmdbAssetCount ?? 0;
        showToast(`同步任务执行成功（云资源 ${cloudCount}，CMDB ${cmdbCount}）`);
      }
      await loadAssetPage(assetPage, assetPageSize, assetQuery, selectedAccountID);
    } catch (error) {
      showToast(extractErrorMessage(error, "同步任务执行失败"));
    } finally {
      setAccountSyncID(null);
    }
  }

  async function handleDeleteAccount(accountID: number) {
    setAccountDeleteID(accountID);
    try {
      await deleteCloudAccount(accountID);
      if (selectedAccountID === accountID) {
        setSelectedAccountID(null);
      }
      await loadAccountPage(accountPage, accountPageSize, accountQuery);
      await loadAssetPage(assetPage, assetPageSize, assetQuery, selectedAccountID === accountID ? null : selectedAccountID);
      showToast("云账号删除成功");
    } catch {
      showToast("云账号删除失败");
    } finally {
      setAccountDeleteID(null);
    }
  }

  function requestDeleteAccount(account: CloudAccountItem) {
    setDeleteAccountTarget(account);
  }

  async function confirmDeleteAccount() {
    if (!deleteAccountTarget) return;
    await handleDeleteAccount(deleteAccountTarget.id);
    setDeleteAccountTarget(null);
  }

  async function handleSubmitAsset(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!assetForm.resourceId.trim()) {
      showToast("resourceId 不能为空");
      return;
    }
    setAssetSubmitting(true);
    try {
      const tags = parseJSONInput(assetForm.tagsJSON, "tags");
      const metadata = parseJSONInput(assetForm.metadataJSON, "metadata");
      const payload = {
        provider: assetForm.provider,
        accountId: parseOptionalNumber(assetForm.accountId) ?? 0,
        region: assetForm.region.trim(),
        type: assetForm.type,
        resourceId: assetForm.resourceId.trim(),
        name: assetForm.name.trim(),
        status: assetForm.status.trim(),
        source: assetForm.source.trim(),
        expiresAt: assetForm.expiresAt.trim() || undefined,
        tags,
        metadata,
      };
      if (assetEditID) {
        await updateCloudAsset(assetEditID, payload);
      } else {
        await createCloudAsset(payload);
      }
      closeDrawer();
      await loadAssetPage(assetPage, assetPageSize, assetQuery, selectedAccountID);
      showToast("云资源保存成功");
    } catch (error) {
      if (error instanceof Error) {
        showToast(error.message);
      } else {
        showToast("云资源保存失败");
      }
    } finally {
      setAssetSubmitting(false);
    }
  }

  async function handleDeleteAsset(assetID: number) {
    setAssetDeleteID(assetID);
    try {
      await deleteCloudAsset(assetID);
      await loadAssetPage(assetPage, assetPageSize, assetQuery, selectedAccountID);
      showToast("云资源删除成功");
    } catch {
      showToast("云资源删除失败");
    } finally {
      setAssetDeleteID(null);
    }
  }

  function requestDeleteAsset(asset: CloudAssetItem) {
    if (isRunningStatus(asset.status)) {
      showToast("运行中资源不可删除");
      return;
    }
    setDeleteAssetTarget(asset);
  }

  async function confirmDeleteAsset() {
    if (!deleteAssetTarget) return;
    await handleDeleteAsset(deleteAssetTarget.id);
    setDeleteAssetTarget(null);
  }

  function handleAssetFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAssetPage(1);
    setAssetQuery({ ...assetFilter });
  }

  function handleAccountFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAccountPage(1);
    setAccountQuery({ ...accountFilter });
  }

  function handleAccountFilterReset() {
    const initial = defaultCloudAccountFilter();
    setAccountFilter(initial);
    setAccountQuery(initial);
    setAccountPage(1);
  }

  function handleAssetFilterReset() {
    const initial = defaultCloudAssetFilter();
    setAssetFilter(initial);
    setAssetQuery(initial);
    setAssetPage(1);
    setSelectedAccountID(null);
  }

  const drawerVisible = drawer.type !== "closed";
  const showAccountForm = drawer.type === "account-create" || drawer.type === "account-edit";
  const showAssetForm = drawer.type === "asset-create" || drawer.type === "asset-edit";
  const isTencentAccountProvider = accountForm.provider === "tencent";

  function drawerTitle(): string {
    if (drawer.type === "account-create") return "创建云账号";
    if (drawer.type === "account-edit") return "编辑云账号";
    if (drawer.type === "asset-create") return "创建云资源";
    if (drawer.type === "asset-edit") return "编辑云资源";
    return "";
  }

  return (
    <section className="page">
      <h2>多云管理</h2>
      <div className="rbac-module-grid">
        <article className="card rbac-module-card cloud-module-card">
          <header className="rbac-module-header">
            <div>
              <h3>云账号管理</h3>
              <p className="muted">统一管理云账号、校验凭据并触发基础资源同步。</p>
            </div>
            <PermissionButton
              permissionKey="button.cloud.account.create"
              className="btn primary cursor-pointer"
              type="button"
              onClick={openAccountCreateDrawer}
            >
              创建账号
            </PermissionButton>
          </header>
          <form className="cloud-filter-bar" onSubmit={handleAccountFilterSubmit}>
            <input
              className="cloud-filter-control cloud-filter-keyword"
              aria-label="账号关键词"
              value={accountFilter.keyword}
              onChange={(event) => setAccountFilter((prev) => ({ ...prev, keyword: event.target.value }))}
              placeholder="关键词：名称/AccessKey"
            />
            <select
              className="cloud-filter-control"
              aria-label="账号云厂商"
              value={accountFilter.provider}
              onChange={(event) => setAccountFilter((prev) => ({ ...prev, provider: event.target.value }))}
            >
              <option value="">云厂商：全部</option>
              {providerOptions.map((provider) => (
                <option key={provider} value={provider}>{provider}</option>
              ))}
            </select>
            <input
              className="cloud-filter-control"
              aria-label="账号地域"
              value={accountFilter.region}
              onChange={(event) => setAccountFilter((prev) => ({ ...prev, region: event.target.value }))}
              placeholder="地域：ap-southeast-1"
            />
            <select
              className="cloud-filter-control"
              aria-label="账号校验状态"
              value={accountFilter.verified}
              onChange={(event) => setAccountFilter((prev) => ({ ...prev, verified: event.target.value }))}
            >
              <option value="">校验状态：全部</option>
              <option value="true">已校验</option>
              <option value="false">未校验</option>
            </select>
            <div className="cloud-filter-actions">
              <button className="btn cursor-pointer" type="submit" disabled={accountLoading}>查询</button>
              <button className="btn cursor-pointer" type="button" onClick={handleAccountFilterReset}>重置</button>
            </div>
          </form>
          <div className="rbac-table-wrapper rbac-module-scroll">
            <table className="rbac-table">
              <thead>
                <tr>
                  {accountVisibleColumnSet.has("id") && <th>ID</th>}
                  {accountVisibleColumnSet.has("provider") && <th>云厂商</th>}
                  {accountVisibleColumnSet.has("name") && <th>名称</th>}
                  {accountVisibleColumnSet.has("region") && <th>地域</th>}
                  {accountVisibleColumnSet.has("verified") && <th>校验状态</th>}
                  {accountVisibleColumnSet.has("updatedAt") && <th>更新时间</th>}
                  {accountVisibleColumnSet.has("actions") && (
                    <th>
                      <div className="table-actions-header">
                        <span>操作</span>
                        <button
                          className="table-settings-trigger cursor-pointer"
                          type="button"
                          onClick={() => setTableSettingsTarget("accounts")}
                          aria-label="云账号列表设置"
                        >
                          ⚙️
                        </button>
                      </div>
                    </th>
                  )}
                </tr>
              </thead>
              <tbody>
                {accountLoading
                  ? <tr><td colSpan={accountColSpan}>加载中...</td></tr>
                  : accounts.length === 0
                    ? <tr><td colSpan={accountColSpan}>暂无数据</td></tr>
                    : accounts.map((item) => (
                      <tr key={item.id}>
                        {accountVisibleColumnSet.has("id") && <td>{item.id}</td>}
                        {accountVisibleColumnSet.has("provider") && <td>{item.provider}</td>}
                        {accountVisibleColumnSet.has("name") && <td>{item.name}</td>}
                        {accountVisibleColumnSet.has("region") && <td>{item.region || "-"}</td>}
                        {accountVisibleColumnSet.has("verified") && <td>{item.isVerified ? "已校验" : "未校验"}</td>}
                        {accountVisibleColumnSet.has("updatedAt") && <td>{formatDateTime(item.updatedAt)}</td>}
                        {accountVisibleColumnSet.has("actions") && (
                          <td>
                            <div className="rbac-row-actions">
                              <RowActionOverflow
                                title="云账号更多操作"
                                actions={[
                                  {
                                    key: `${item.id}-verify`,
                                    label: accountVerifyID === item.id ? "校验中..." : "校验",
                                    permissionKey: "button.cloud.account.verify",
                                    disabled: accountVerifyID === item.id,
                                    onClick: () => void handleVerifyAccount(item.id),
                                  },
                                  {
                                    key: `${item.id}-sync`,
                                    label: accountSyncID === item.id ? "同步中..." : "同步",
                                    permissionKey: "button.cloud.account.sync",
                                    disabled: accountSyncID === item.id,
                                    onClick: () => void handleSyncAccount(item.id),
                                  },
                                  {
                                    key: `${item.id}-assets`,
                                    label: "查看资源",
                                    onClick: () => {
                                      setSelectedAccountID(item.id);
                                      setAssetPage(1);
                                    },
                                  },
                                  {
                                    key: `${item.id}-edit`,
                                    label: "编辑",
                                    permissionKey: "button.cloud.account.update",
                                    onClick: () => openAccountEditDrawer(item),
                                  },
                                  {
                                    key: `${item.id}-delete`,
                                    label: accountDeleteID === item.id ? "删除中..." : "删除",
                                    permissionKey: "button.cloud.account.delete",
                                    disabled: accountDeleteID === item.id,
                                    onClick: () => requestDeleteAccount(item),
                                  },
                                ]}
                              />
                            </div>
                          </td>
                        )}
                      </tr>
                    ))}
              </tbody>
            </table>
          </div>
          <footer className="rbac-pagination">
            <div className="rbac-pagination-group">
              <span>总计 {accountTotal} 条</span>
              <select
                className="rbac-pagination-select cursor-pointer"
                value={accountPageSize}
                onChange={(event) => {
                  setAccountPageSize(Number(event.target.value));
                  setAccountPage(1);
                }}
              >
                {pageSizeOptions.map((option) => (
                  <option key={option} value={option}>{option}/页</option>
                ))}
              </select>
            </div>
            <div className="rbac-pagination-group">
              <button className="btn cursor-pointer" type="button" disabled={accountPage <= 1} onClick={() => setAccountPage((page) => Math.max(1, page - 1))}>上一页</button>
              <span className="rbac-pagination-text">{accountPage} / {accountTotalPages}</span>
              <button className="btn cursor-pointer" type="button" disabled={accountPage >= accountTotalPages} onClick={() => setAccountPage((page) => Math.min(accountTotalPages, page + 1))}>下一页</button>
            </div>
          </footer>
        </article>

        <article className="card rbac-module-card cloud-module-card">
          <header className="rbac-module-header">
            <div>
              <h3>云资源管理</h3>
              <p className="muted">统一管理多云基础资源，支持 CRUD 与按账号/地域筛选。</p>
            </div>
            <PermissionButton
              permissionKey="button.cloud.asset.create"
              className="btn primary cursor-pointer"
              type="button"
              onClick={openAssetCreateDrawer}
            >
              创建资源
            </PermissionButton>
          </header>
          <form className="cloud-filter-bar" onSubmit={handleAssetFilterSubmit}>
            <input
              className="cloud-filter-control cloud-filter-keyword"
              id="cloud-asset-filter-keyword"
              aria-label="关键词"
              value={assetFilter.keyword}
              onChange={(event) => setAssetFilter((prev) => ({ ...prev, keyword: event.target.value }))}
              placeholder="关键词：名称/resourceId"
            />
            <select
              className="cloud-filter-control"
              id="cloud-asset-filter-provider"
              aria-label="云厂商"
              value={assetFilter.provider}
              onChange={(event) => setAssetFilter((prev) => ({ ...prev, provider: event.target.value }))}
            >
              <option value="">云厂商：全部</option>
              {providerOptions.map((provider) => (
                <option key={provider} value={provider}>{provider}</option>
              ))}
            </select>
            <select
              className="cloud-filter-control"
              id="cloud-asset-filter-account"
              aria-label="云账号"
              value={assetFilter.accountId}
              onChange={(event) => setAssetFilter((prev) => ({ ...prev, accountId: event.target.value }))}
              disabled={selectedAccountID !== null}
            >
              <option value="">云账号：全部</option>
              {accounts.map((account) => (
                <option key={account.id} value={String(account.id)}>
                  {account.id} / {account.name}
                </option>
              ))}
            </select>
            <input
              className="cloud-filter-control"
              id="cloud-asset-filter-region"
              aria-label="地域"
              value={assetFilter.region}
              onChange={(event) => setAssetFilter((prev) => ({ ...prev, region: event.target.value }))}
              placeholder="地域：ap-southeast-1"
            />
            <select
              className="cloud-filter-control"
              id="cloud-asset-filter-type"
              aria-label="资源类型"
              value={assetFilter.type}
              onChange={(event) => setAssetFilter((prev) => ({ ...prev, type: event.target.value }))}
            >
              <option value="">资源类型：全部</option>
              {cloudAssetTypeOptions.map((item) => (
                <option key={item} value={item}>{item}</option>
              ))}
            </select>
            <div className="cloud-filter-actions">
              <button className="btn cursor-pointer" type="submit" disabled={assetLoading}>查询</button>
              <button className="btn cursor-pointer" type="button" onClick={handleAssetFilterReset}>重置</button>
              {selectedAccountID ? (
                <button className="btn cursor-pointer" type="button" onClick={() => setSelectedAccountID(null)}>取消账号过滤</button>
              ) : null}
            </div>
          </form>

          <div className="rbac-table-wrapper rbac-module-scroll">
            <table className="rbac-table">
              <thead>
                <tr>
                  {assetVisibleColumnSet.has("id") && <th>ID</th>}
                  {assetVisibleColumnSet.has("providerAccount") && <th>provider/account</th>}
                  {assetVisibleColumnSet.has("type") && <th>类型</th>}
                  {assetVisibleColumnSet.has("resourceId") && <th>resourceId</th>}
                  {assetVisibleColumnSet.has("name") && <th>名称</th>}
                  {assetVisibleColumnSet.has("status") && <th>状态</th>}
                  {assetVisibleColumnSet.has("source") && <th>来源</th>}
                  {assetVisibleColumnSet.has("expiresAt") && <th>过期时间</th>}
                  {assetVisibleColumnSet.has("lastSyncedAt") && <th>最近同步</th>}
                  {assetVisibleColumnSet.has("actions") && (
                    <th>
                      <div className="table-actions-header">
                        <span>操作</span>
                        <button
                          className="table-settings-trigger cursor-pointer"
                          type="button"
                          onClick={() => setTableSettingsTarget("assets")}
                          aria-label="云资源列表设置"
                        >
                          ⚙️
                        </button>
                      </div>
                    </th>
                  )}
                </tr>
              </thead>
              <tbody>
                {assetLoading
                  ? <tr><td colSpan={assetColSpan}>加载中...</td></tr>
                  : assets.length === 0
                    ? <tr><td colSpan={assetColSpan}>暂无数据</td></tr>
                    : assets.map((item) => (
                      <tr key={item.id}>
                        {assetVisibleColumnSet.has("id") && <td>{item.id}</td>}
                        {assetVisibleColumnSet.has("providerAccount") && <td>{item.provider}/{item.accountId || "-"}</td>}
                        {assetVisibleColumnSet.has("type") && <td>{item.type}</td>}
                        {assetVisibleColumnSet.has("resourceId") && <td>{item.resourceId}</td>}
                        {assetVisibleColumnSet.has("name") && <td>{item.name}</td>}
                        {assetVisibleColumnSet.has("status") && <td>{item.status || "-"}</td>}
                        {assetVisibleColumnSet.has("source") && <td>{item.source || "-"}</td>}
                        {assetVisibleColumnSet.has("expiresAt") && <td>{formatDateTime(item.expiresAt)}</td>}
                        {assetVisibleColumnSet.has("lastSyncedAt") && <td>{formatDateTime(item.lastSyncedAt)}</td>}
                        {assetVisibleColumnSet.has("actions") && (
                          <td>
                            <div className="rbac-row-actions">
                              <RowActionOverflow
                                title="云资源更多操作"
                                actions={[
                                  {
                                    key: `${item.id}-edit`,
                                    label: "编辑",
                                    permissionKey: "button.cloud.asset.update",
                                    onClick: () => openAssetEditDrawer(item),
                                  },
                                  {
                                    key: `${item.id}-delete`,
                                    label: assetDeleteID === item.id ? "删除中..." : "删除",
                                    permissionKey: "button.cloud.asset.delete",
                                    disabled: assetDeleteID === item.id || isRunningStatus(item.status),
                                    onClick: () => requestDeleteAsset(item),
                                  },
                                ]}
                              />
                            </div>
                          </td>
                        )}
                      </tr>
                    ))}
              </tbody>
            </table>
          </div>
          <footer className="rbac-pagination">
            <div className="rbac-pagination-group">
              <span>总计 {assetTotal} 条</span>
              <select
                className="rbac-pagination-select cursor-pointer"
                value={assetPageSize}
                onChange={(event) => {
                  setAssetPageSize(Number(event.target.value));
                  setAssetPage(1);
                }}
              >
                {pageSizeOptions.map((option) => (
                  <option key={option} value={option}>{option}/页</option>
                ))}
              </select>
            </div>
            <div className="rbac-pagination-group">
              <button className="btn cursor-pointer" type="button" disabled={assetPage <= 1} onClick={() => setAssetPage((page) => Math.max(1, page - 1))}>上一页</button>
              <span className="rbac-pagination-text">{assetPage} / {assetTotalPages}</span>
              <button className="btn cursor-pointer" type="button" disabled={assetPage >= assetTotalPages} onClick={() => setAssetPage((page) => Math.min(assetTotalPages, page + 1))}>下一页</button>
            </div>
          </footer>
        </article>
      </div>

      <TableSettingsModal
        open={tableSettingsTarget === "accounts"}
        title="云账号列表设置"
        columns={cloudAccountTableColumns}
        visibleColumnKeys={visibleAccountColumnKeys}
        onToggleColumn={toggleAccountVisibleColumn}
        onReset={() => setVisibleAccountColumnKeys(sanitizeVisibleColumnKeys(defaultAccountVisibleColumnKeys, cloudAccountTableColumns))}
        onClose={() => setTableSettingsTarget("closed")}
      />

      <TableSettingsModal
        open={tableSettingsTarget === "assets"}
        title="云资源列表设置"
        columns={cloudAssetTableColumns}
        visibleColumnKeys={visibleAssetColumnKeys}
        onToggleColumn={toggleAssetVisibleColumn}
        onReset={() => setVisibleAssetColumnKeys(sanitizeVisibleColumnKeys(defaultAssetVisibleColumnKeys, cloudAssetTableColumns))}
        onClose={() => setTableSettingsTarget("closed")}
      />

      {drawerVisible && (
        <div className="rbac-drawer-mask" onClick={closeDrawer}>
          <aside className="rbac-drawer" onClick={(event) => event.stopPropagation()}>
            <header className="rbac-drawer-header">
              <h3>{drawerTitle()}</h3>
              <button className="btn ghost cursor-pointer" type="button" onClick={closeDrawer}>
                关闭
              </button>
            </header>
            <div className="rbac-drawer-body">
              {showAccountForm && (
                <form className="form-grid" onSubmit={handleSubmitAccount}>
                  <label htmlFor="cloud-account-provider">云厂商</label>
                  <select
                    id="cloud-account-provider"
                    value={accountForm.provider}
                    onChange={(event) => setAccountForm((prev) => ({ ...prev, provider: event.target.value as CloudProvider }))}
                  >
                    {providerOptions.map((provider) => (
                      <option key={provider} value={provider}>{provider}</option>
                    ))}
                  </select>
                  <label htmlFor="cloud-account-name">账号名称</label>
                  <input
                    id="cloud-account-name"
                    value={accountForm.name}
                    onChange={(event) => setAccountForm((prev) => ({ ...prev, name: event.target.value }))}
                    placeholder="生产账号"
                  />
                  <label htmlFor="cloud-account-ak">{isTencentAccountProvider ? "AccessKey（SecretId）" : "AccessKey"}</label>
                  <input
                    id="cloud-account-ak"
                    value={accountForm.accessKey}
                    onChange={(event) => setAccountForm((prev) => ({ ...prev, accessKey: event.target.value }))}
                    placeholder={accountEditID ? "留空保持不变" : (isTencentAccountProvider ? "SecretId (AKID...)" : "AccessKey")}
                  />
                  <label htmlFor="cloud-account-sk">{isTencentAccountProvider ? "SecretKey（腾讯云）" : "SecretKey"}</label>
                  <input
                    id="cloud-account-sk"
                    value={accountForm.secretKey}
                    onChange={(event) => setAccountForm((prev) => ({ ...prev, secretKey: event.target.value }))}
                    placeholder={accountEditID ? "留空保持不变" : (isTencentAccountProvider ? "腾讯云 SecretKey" : "SecretKey")}
                  />
                  {isTencentAccountProvider && (
                    <>
                      <span className="muted">腾讯云需填写 SecretId/SecretKey；开发环境可用 `mock_` 前缀触发模拟。</span>
                      <span />
                    </>
                  )}
                  <label htmlFor="cloud-account-region">默认地域</label>
                  <input
                    id="cloud-account-region"
                    value={accountForm.region}
                    onChange={(event) => setAccountForm((prev) => ({ ...prev, region: event.target.value }))}
                    placeholder={isTencentAccountProvider ? "ap-guangzhou" : "ap-southeast-1"}
                  />
                  <PermissionButton
                    permissionKey={accountEditID ? "button.cloud.account.update" : "button.cloud.account.create"}
                    className="btn primary cursor-pointer"
                    type="submit"
                    disabled={accountSubmitting}
                  >
                    {accountSubmitting ? "保存中..." : "保存"}
                  </PermissionButton>
                </form>
              )}

              {showAssetForm && (
                <form className="form-grid" onSubmit={handleSubmitAsset}>
                  <label htmlFor="cloud-asset-provider">云厂商</label>
                  <select
                    id="cloud-asset-provider"
                    value={assetForm.provider}
                    onChange={(event) => setAssetForm((prev) => ({ ...prev, provider: event.target.value as CloudProvider }))}
                  >
                    {providerOptions.map((provider) => (
                      <option key={provider} value={provider}>{provider}</option>
                    ))}
                  </select>
                  <label htmlFor="cloud-asset-account-id">accountId（可选）</label>
                  <input
                    id="cloud-asset-account-id"
                    value={assetForm.accountId}
                    onChange={(event) => setAssetForm((prev) => ({ ...prev, accountId: event.target.value }))}
                    placeholder="1"
                  />
                  <label htmlFor="cloud-asset-region">region</label>
                  <input
                    id="cloud-asset-region"
                    value={assetForm.region}
                    onChange={(event) => setAssetForm((prev) => ({ ...prev, region: event.target.value }))}
                    placeholder="ap-southeast-1"
                  />
                  <label htmlFor="cloud-asset-type">类型</label>
                  <select
                    id="cloud-asset-type"
                    value={assetForm.type}
                    onChange={(event) => {
                      const nextType = event.target.value as CloudAssetType;
                      setAssetForm((prev) => ({ ...prev, type: nextType }));
                      setSelectedAssetTemplate(normalizeCloudAssetType(nextType) ?? "custom");
                    }}
                  >
                    {cloudAssetTypeOptions.map((item) => (
                      <option key={item} value={item}>{item}</option>
                    ))}
                  </select>
                  <label htmlFor="cloud-asset-template">模板示例</label>
                  <div className="rbac-actions">
                    <select
                      id="cloud-asset-template"
                      value={selectedAssetTemplate}
                      onChange={(event) => setSelectedAssetTemplate(event.target.value as AssetTemplateKey)}
                    >
                      <option value="custom">不使用模板</option>
                      {cloudAssetTypeOptions.map((item) => (
                        <option key={item} value={item}>{cloudAssetTemplateMap[item]?.label ?? `${item} 模板`}</option>
                      ))}
                    </select>
                    <button
                      className="btn cursor-pointer"
                      type="button"
                      disabled={selectedAssetTemplate === "custom"}
                      onClick={handleApplyAssetTemplate}
                    >
                      填充 tags/metadata
                    </button>
                  </div>
                  <label htmlFor="cloud-asset-resource-id">resourceId</label>
                  <input
                    id="cloud-asset-resource-id"
                    value={assetForm.resourceId}
                    onChange={(event) => setAssetForm((prev) => ({ ...prev, resourceId: event.target.value }))}
                    placeholder="i-xxxxxxxx"
                  />
                  <label htmlFor="cloud-asset-name">名称</label>
                  <input
                    id="cloud-asset-name"
                    value={assetForm.name}
                    onChange={(event) => setAssetForm((prev) => ({ ...prev, name: event.target.value }))}
                    placeholder="ecs-prod-01"
                  />
                  <label htmlFor="cloud-asset-status">状态</label>
                  <input
                    id="cloud-asset-status"
                    value={assetForm.status}
                    onChange={(event) => setAssetForm((prev) => ({ ...prev, status: event.target.value }))}
                    placeholder="active"
                  />
                  <label htmlFor="cloud-asset-source">来源</label>
                  <input
                    id="cloud-asset-source"
                    value={assetForm.source}
                    onChange={(event) => setAssetForm((prev) => ({ ...prev, source: event.target.value }))}
                    placeholder="CloudSync"
                  />
                  <label htmlFor="cloud-asset-expires-at">过期时间</label>
                  <input
                    id="cloud-asset-expires-at"
                    value={assetForm.expiresAt}
                    onChange={(event) => setAssetForm((prev) => ({ ...prev, expiresAt: event.target.value }))}
                    placeholder="RFC3339"
                  />
                  <label htmlFor="cloud-asset-tags">tags JSON</label>
                  <textarea
                    id="cloud-asset-tags"
                    value={assetForm.tagsJSON}
                    onChange={(event) => setAssetForm((prev) => ({ ...prev, tagsJSON: event.target.value }))}
                    placeholder='{"env":"prod"}'
                  />
                  <label htmlFor="cloud-asset-metadata">metadata JSON</label>
                  <textarea
                    id="cloud-asset-metadata"
                    value={assetForm.metadataJSON}
                    onChange={(event) => setAssetForm((prev) => ({ ...prev, metadataJSON: event.target.value }))}
                    placeholder='{"cpu":"4"}'
                  />
                  <PermissionButton
                    permissionKey={assetEditID ? "button.cloud.asset.update" : "button.cloud.asset.create"}
                    className="btn primary cursor-pointer"
                    type="submit"
                    disabled={assetSubmitting}
                  >
                    {assetSubmitting ? "保存中..." : "保存"}
                  </PermissionButton>
                </form>
              )}
            </div>
          </aside>
        </div>
      )}

      <DeleteConfirmModal
        open={deleteAccountTarget !== null}
        title="删除云账号确认"
        description={`将删除云账号：${deleteAccountTarget?.name || "-"}（${deleteAccountTarget?.provider || "-"}）`}
        confirming={deleteAccountTarget !== null && accountDeleteID === deleteAccountTarget.id}
        onCancel={() => setDeleteAccountTarget(null)}
        onConfirm={() => void confirmDeleteAccount()}
      />

      <DeleteConfirmModal
        open={deleteAssetTarget !== null}
        title="删除云资源确认"
        description={`将删除云资源：${deleteAssetTarget?.name || deleteAssetTarget?.resourceId || "-"}`}
        confirming={deleteAssetTarget !== null && assetDeleteID === deleteAssetTarget.id}
        onCancel={() => setDeleteAssetTarget(null)}
        onConfirm={() => void confirmDeleteAsset()}
      />
    </section>
  );
}

function parseOptionalNumber(raw: string): number | undefined {
  const text = raw.trim();
  if (!text) return undefined;
  const value = Number(text);
  if (Number.isNaN(value) || value < 0) return undefined;
  return value;
}

function normalizeProvider(raw: string): CloudProvider | undefined {
  if (providerOptions.includes(raw as CloudProvider)) {
    return raw as CloudProvider;
  }
  return undefined;
}

function normalizeCloudAssetType(raw: string): CloudAssetType | undefined {
  if (cloudAssetTypeOptions.includes(raw as CloudAssetType)) {
    return raw as CloudAssetType;
  }
  return undefined;
}

function parseJSONInput(raw: string, fieldName: string): Record<string, unknown> {
  const text = raw.trim();
  if (!text) return {};
  try {
    const parsed = JSON.parse(text);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
    throw new Error(`${fieldName} 必须是 JSON 对象`);
  } catch {
    throw new Error(`${fieldName} 不是合法 JSON 对象`);
  }
}

function totalPages(total: number, pageSize: number): number {
  if (pageSize <= 0) return 1;
  return Math.max(1, Math.ceil(total / pageSize));
}

function formatDateTime(value?: string): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function isLikelyTencentSecretId(value: string): boolean {
  return value.trim().toUpperCase().startsWith("AKID");
}

function extractErrorMessage(error: unknown, fallback: string): string {
  if (!error || typeof error !== "object") return fallback;
  const message = (error as { response?: { data?: { message?: unknown } } }).response?.data?.message;
  if (typeof message === "string" && message.trim()) return message.trim();
  const generic = (error as { message?: unknown }).message;
  if (typeof generic === "string" && generic.trim()) return generic.trim();
  return fallback;
}

function isRunningStatus(status?: string): boolean {
  const value = (status ?? "").trim().toLowerCase();
  return value === "running" || value === "运行中";
}
