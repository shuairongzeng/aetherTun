import type { BasicProxySettings } from "../types";

type FirstRunOnboardingStep = "welcome" | "config";

type FirstRunOnboardingProps = {
  step: FirstRunOnboardingStep;
  value: BasicProxySettings;
  errors: Partial<Record<keyof BasicProxySettings, string>>;
  saving: boolean;
  statusText: string;
  onChange: (next: BasicProxySettings) => void;
  onStart: () => void;
  onBack: () => void;
  onSkip: () => void;
  onSave: () => void | Promise<void>;
};

export function FirstRunOnboarding(props: FirstRunOnboardingProps) {
  return (
    <div className="onboarding-overlay" role="dialog" aria-modal="true" aria-labelledby="onboarding-title">
      <article className="onboarding-card">
        {props.step === "welcome" ? (
          <div className="onboarding-shell">
            <div>
              <p className="eyebrow">首次启动引导</p>
              <h2 id="onboarding-title">欢迎使用 Aether</h2>
              <p className="onboarding-copy">
                Aether 需要一个可用的上游代理后才能正常工作。先完成基础代理配置，再点击“启动代理”会更顺手。
              </p>

              <div className="onboarding-checklist">
                <article className="onboarding-checklist__item">只需准备一组可用的上游代理参数</article>
                <article className="onboarding-checklist__item">保存后就能回到主界面继续启动代理</article>
                <article className="onboarding-checklist__item">即使暂时跳过，主界面也会继续提醒你</article>
              </div>

              <div className="onboarding-actions">
                <button className="primary-button" type="button" onClick={props.onStart}>
                  开始配置
                </button>
                <button className="secondary-button secondary-button--ghost" type="button" onClick={props.onSkip}>
                  暂时跳过
                </button>
              </div>
            </div>

            <aside className="onboarding-side">
              <span className="onboarding-side__badge">推荐先完成配置</span>
              <h3 className="onboarding-side__title">你将得到什么</h3>
              <ul className="onboarding-side__list">
                <li>启动按钮、日志区、配置区会保持同一条操作链路。</li>
                <li>保存后无需自己找配置文件路径，新手也能快速上手。</li>
                <li>后续若要高级调整，仍然可以随时打开正式配置文件。</li>
              </ul>
            </aside>
          </div>
        ) : (
          <div className="onboarding-shell">
            <div>
              <p className="eyebrow">首次启动引导</p>
              <h2 id="onboarding-title">填写基础代理配置</h2>
              <p className="onboarding-copy">只需要填好代理地址、端口和类型，就可以回到主界面继续启动代理。</p>

              <div className="config-form-grid onboarding-form-grid">
                <label className="config-field" htmlFor="onboarding-proxy-host">
                  <span className="config-field__label">代理地址</span>
                  <span className="config-field__hint">优先填写固定的本机地址或局域网可访问地址。</span>
                  <input
                    id="onboarding-proxy-host"
                    className="config-input"
                    type="text"
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

                <label className="config-field" htmlFor="onboarding-proxy-port">
                  <span className="config-field__label">代理端口</span>
                  <span className="config-field__hint">保持与上游客户端监听端口一致。</span>
                  <input
                    id="onboarding-proxy-port"
                    className="config-input"
                    type="number"
                    min={1}
                    max={65535}
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

                <label className="config-field" htmlFor="onboarding-proxy-type">
                  <span className="config-field__label">代理类型</span>
                  <span className="config-field__hint">确保与上游客户端实际提供的协议一致。</span>
                  <select
                    id="onboarding-proxy-type"
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

              {props.statusText ? (
                <div className="config-status-message config-status-message--info" role="status" aria-live="polite">
                  <strong className="config-status-message__title">引导提示</strong>
                  <span>{props.statusText}</span>
                </div>
              ) : null}

              <div className="onboarding-actions">
                <button className="secondary-button secondary-button--ghost" type="button" onClick={props.onBack}>
                  返回
                </button>
                <button className="primary-button" type="button" onClick={() => void props.onSave()}>
                  {props.saving ? "保存中…" : "保存并进入主界面"}
                </button>
              </div>
            </div>

            <aside className="onboarding-side">
              <span className="onboarding-side__badge">推荐填写方式</span>
              <h3 className="onboarding-side__title">推荐填写方式</h3>
              <ul className="onboarding-side__list">
                <li>如果你的上游代理跑在本机，常见地址是 `127.0.0.1`。</li>
                <li>端口通常与上游客户端界面里显示的监听端口一致。</li>
                <li>协议不确定时，优先回到客户端确认是 SOCKS5 还是 HTTP。</li>
              </ul>
            </aside>
          </div>
        )}
      </article>
    </div>
  );
}
