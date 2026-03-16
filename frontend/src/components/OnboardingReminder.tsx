type OnboardingReminderProps = {
  onContinue: () => void;
  onOpenConfigFile: () => void | Promise<void>;
};

export function OnboardingReminder(props: OnboardingReminderProps) {
  return (
    <section className="onboarding-reminder" aria-label="首次配置提醒">
      <div>
        <span className="onboarding-reminder__badge">继续配置更推荐</span>
        <strong className="onboarding-reminder__title">尚未完成首次代理配置</strong>
        <p className="onboarding-reminder__text">建议先填写代理地址、端口和类型，再启动代理。</p>
      </div>
      <div className="onboarding-reminder__actions">
        <button className="secondary-button" type="button" onClick={props.onContinue}>
          继续配置
        </button>
        <button className="secondary-button secondary-button--ghost" type="button" onClick={() => void props.onOpenConfigFile()}>
          打开配置文件
        </button>
      </div>
    </section>
  );
}
