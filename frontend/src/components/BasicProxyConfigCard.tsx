import type { BasicProxySettings, BasicProxyStatus } from "../types";

type BasicProxyConfigCardProps = {
  value: BasicProxySettings;
  dirty: boolean;
  saving: boolean;
  errors: Partial<Record<keyof BasicProxySettings, string>>;
  status: BasicProxyStatus;
  onChange: (next: BasicProxySettings) => void;
  onSave: () => void | Promise<void>;
  onResetDefaults: () => void | Promise<void>;
  onOpenConfigFile: () => void | Promise<void>;
};

function hasErrors(errors: BasicProxyConfigCardProps["errors"]) {
  return Object.values(errors).some(Boolean);
}

function statusTitle(tone: BasicProxyStatus["tone"]) {
  switch (tone) {
    case "success":
      return "保存成功";
    case "error":
      return "保存失败";
    default:
      return "生效提示";
  }
}

export function BasicProxyConfigCard(props: BasicProxyConfigCardProps) {
  const saveDisabled = !props.dirty || props.saving || hasErrors(props.errors);
  const endpointLabel = `${props.value.host}:${props.value.port}`;

  return (
    <div className="card config-card">
      <div className="config-card__header">
        <h2 className="config-card__title">代理配置</h2>
        <span className={`config-card__badge config-card__badge--${props.dirty ? "dirty" : "clean"}`}>
          {props.dirty ? "有未保存修改" : "已保存"}
        </span>
      </div>

      <div className="config-overview">
        <div className="config-overview__item">
          <span className="config-overview__label">当前上游</span>
          <strong className="config-overview__value">{endpointLabel}</strong>
        </div>
        <div className="config-overview__item">
          <span className="config-overview__label">协议</span>
          <strong className="config-overview__value">{props.value.type.toUpperCase()}</strong>
        </div>
      </div>

      <div className="config-form">
        <label className="config-field" htmlFor="proxy-host">
          <span className="config-field__label">代理地址</span>
          <span className="config-field__hint">支持 IP、localhost 或局域网域名</span>
          <input
            id="proxy-host"
            className="config-input"
            type="text"
            placeholder="127.0.0.1"
            value={props.value.host}
            onChange={(event) =>
              props.onChange({
                ...props.value,
                host: event.target.value
              })
            }
          />
          {props.errors.host ? <span className="config-field__error">{props.errors.host}</span> : null}
        </label>

        <label className="config-field" htmlFor="proxy-port">
          <span className="config-field__label">代理端口</span>
          <span className="config-field__hint">请填写 1-65535 范围内的端口</span>
          <input
            id="proxy-port"
            className="config-input"
            type="number"
            min={1}
            max={65535}
            placeholder="10808"
            value={props.value.port}
            onChange={(event) =>
              props.onChange({
                ...props.value,
                port: Number.parseInt(event.target.value, 10) || 0
              })
            }
          />
          {props.errors.port ? <span className="config-field__error">{props.errors.port}</span> : null}
        </label>

        <label className="config-field" htmlFor="proxy-type">
          <span className="config-field__label">代理类型</span>
          <span className="config-field__hint">选择与上游客户端一致的协议类型</span>
          <select
            id="proxy-type"
            className="config-input"
            value={props.value.type}
            onChange={(event) =>
              props.onChange({
                ...props.value,
                type: event.target.value
              })
            }
          >
            <option value="socks5">SOCKS5</option>
            <option value="http">HTTP</option>
          </select>
          {props.errors.type ? <span className="config-field__error">{props.errors.type}</span> : null}
        </label>
      </div>

      {props.status.text ? (
        <div
          className={`config-status-message config-status-message--${props.status.tone}`}
          role="status"
          aria-live="polite"
        >
          <strong className="config-status-message__title">{statusTitle(props.status.tone)}</strong>
          <span>{props.status.text}</span>
        </div>
      ) : null}

      <div className="config-actions">
        <button
          className="btn-primary config-actions__primary"
          type="button"
          disabled={saveDisabled}
          onClick={() => void props.onSave()}
        >
          {props.saving ? "保存中…" : "保存配置"}
        </button>

        <div className="config-actions__secondary">
          <button className="btn-secondary" type="button" onClick={() => void props.onResetDefaults()}>
            恢复默认值
          </button>
        </div>
      </div>

      <div className="config-footer">
        <p className="config-footer__note">运行中修改配置时，界面会提示是否立即重启代理。</p>
        <p className="config-footer__link">
          需要修改高级网络配置？{" "}
          <button type="button" onClick={() => void props.onOpenConfigFile()}>
            打开配置文件 →
          </button>
        </p>
      </div>
    </div>
  );
}
