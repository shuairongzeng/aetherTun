import { fireEvent, render, screen } from "@testing-library/react";
import { vi } from "vitest";
import { FirstRunOnboarding } from "./FirstRunOnboarding";

it("renders the welcome step with start and skip actions", () => {
  render(
    <FirstRunOnboarding
      step="welcome"
      value={{ host: "127.0.0.1", port: 10808, type: "socks5" }}
      errors={{}}
      saving={false}
      statusText=""
      onChange={() => {}}
      onStart={() => {}}
      onBack={() => {}}
      onSkip={() => {}}
      onSave={() => {}}
    />
  );

  expect(screen.getByRole("heading", { name: "欢迎使用 Aether" })).toBeInTheDocument();
  expect(screen.getByText("只需准备一组可用的上游代理参数")).toBeInTheDocument();
  expect(screen.getByText("保存后就能回到主界面继续启动代理")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "开始配置" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "暂时跳过" })).toBeInTheDocument();
  expect(screen.getByText("推荐先完成配置")).toBeInTheDocument();
  expect(screen.getByText("你将得到什么")).toBeInTheDocument();
});

it("renders the config step and submits save action", () => {
  const onSave = vi.fn();

  render(
    <FirstRunOnboarding
      step="config"
      value={{ host: "127.0.0.1", port: 10808, type: "socks5" }}
      errors={{}}
      saving={false}
      statusText=""
      onChange={() => {}}
      onStart={() => {}}
      onBack={() => {}}
      onSkip={() => {}}
      onSave={onSave}
    />
  );

  expect(screen.getByLabelText(/代理地址/)).toBeInTheDocument();
  expect(screen.getByLabelText(/代理端口/)).toBeInTheDocument();
  expect(screen.getByLabelText(/代理类型/)).toBeInTheDocument();
  expect(screen.getAllByText("推荐填写方式").length).toBeGreaterThan(0);

  fireEvent.click(screen.getByRole("button", { name: "保存并进入主界面" }));

  expect(onSave).toHaveBeenCalledTimes(1);
});
