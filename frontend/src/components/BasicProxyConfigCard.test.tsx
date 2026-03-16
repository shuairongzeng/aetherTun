import { render, screen } from "@testing-library/react";
import { vi } from "vitest";
import { BasicProxyConfigCard } from "./BasicProxyConfigCard";

it("disables save until the form becomes dirty and valid", () => {
  render(
    <BasicProxyConfigCard
      value={{ host: "127.0.0.1", port: 10808, type: "socks5" }}
      dirty={false}
      saving={false}
      errors={{}}
      status={{ tone: "info", text: "" }}
      onChange={() => {}}
      onSave={() => {}}
      onResetDefaults={() => {}}
      onOpenConfigFile={vi.fn().mockResolvedValue(undefined)}
    />
  );

  expect(screen.getByText("当前上游")).toBeInTheDocument();
  expect(screen.getByText("127.0.0.1:10808")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled();
});

it("shows inline validation text", () => {
  render(
    <BasicProxyConfigCard
      value={{ host: "", port: 10808, type: "socks5" }}
      dirty
      saving={false}
      errors={{ host: "代理地址不能为空" }}
      status={{ tone: "info", text: "" }}
      onChange={() => {}}
      onSave={() => {}}
      onResetDefaults={() => {}}
      onOpenConfigFile={vi.fn().mockResolvedValue(undefined)}
    />
  );

  expect(screen.getByText("代理地址不能为空")).toBeInTheDocument();
});

it("renders a success banner when config is saved", () => {
  render(
    <BasicProxyConfigCard
      value={{ host: "127.0.0.1", port: 10808, type: "socks5" }}
      dirty={false}
      saving={false}
      errors={{}}
      status={{ tone: "success", text: "配置已保存。" }}
      onChange={() => {}}
      onSave={() => {}}
      onResetDefaults={() => {}}
      onOpenConfigFile={vi.fn().mockResolvedValue(undefined)}
    />
  );

  const banner = screen.getByRole("status");
  expect(banner).toHaveTextContent("配置已保存。");
  expect(banner).toHaveTextContent("保存成功");
  expect(banner.className).toContain("config-status-message--success");
});

it("renders an error banner when saving fails", () => {
  render(
    <BasicProxyConfigCard
      value={{ host: "127.0.0.1", port: 10808, type: "socks5" }}
      dirty={false}
      saving={false}
      errors={{}}
      status={{ tone: "error", text: "保存配置失败，请检查配置文件后重试。" }}
      onChange={() => {}}
      onSave={() => {}}
      onResetDefaults={() => {}}
      onOpenConfigFile={vi.fn().mockResolvedValue(undefined)}
    />
  );

  const banner = screen.getByRole("status");
  expect(banner).toHaveTextContent("保存配置失败，请检查配置文件后重试。");
  expect(banner).toHaveTextContent("保存失败");
  expect(banner.className).toContain("config-status-message--error");
});
