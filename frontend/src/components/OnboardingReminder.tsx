type OnboardingReminderProps = {
  onContinue: () => void;
  onOpenConfigFile: () => void | Promise<void>;
};

export function OnboardingReminder(props: OnboardingReminderProps) {
  return (
    <section className="onboarding-reminder" aria-label="首次配置提醒">
      <div className="onboarding-reminder__content">
        <span className="onboarding-reminder__badge">待配置</span>
        <p className="onboarding-reminder__text">尚未完成首次代理配置，建议先填写代理参数再启动。</p>
      </div>
      <div className="onboarding-reminder__actions">
        <button className="btn-secondary" type="button" onClick={props.onContinue}>
          继续配置
        </button>
        <button className="btn-text" type="button" onClick={() => void props.onOpenConfigFile()}>
          打开配置文件
        </button>
      </div>
    </section>
  );
}
