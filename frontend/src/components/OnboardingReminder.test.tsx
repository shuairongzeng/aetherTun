import { fireEvent, render, screen } from "@testing-library/react";
import { vi } from "vitest";
import { OnboardingReminder } from "./OnboardingReminder";

it("renders reminder actions for continuing onboarding", () => {
  const onContinue = vi.fn();
  const onOpenConfigFile = vi.fn().mockResolvedValue(undefined);

  render(<OnboardingReminder onContinue={onContinue} onOpenConfigFile={onOpenConfigFile} />);

  expect(screen.getByText(/尚未完成首次代理配置/)).toBeInTheDocument();
  expect(screen.getByText("继续配置更推荐")).toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: "继续配置" }));
  expect(onContinue).toHaveBeenCalledTimes(1);

  fireEvent.click(screen.getByRole("button", { name: "打开配置文件" }));
  expect(onOpenConfigFile).toHaveBeenCalledTimes(1);
});
