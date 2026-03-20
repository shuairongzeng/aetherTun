import { render, screen } from "@testing-library/react";
import { StatusCard } from "./StatusCard";

it("renders the running phase", () => {
  render(
    <StatusCard
      status={{
        phase: "running",
        proxyEndpoint: "socks5://127.0.0.1:10808",
        tunAdapterName: "Aether-TUN"
      }}
    />
  );

  expect(screen.getByText("代理运行中")).toBeInTheDocument();
  expect(screen.getByText("后台核心已连接，代理正在运行。")).toBeInTheDocument();
  expect(screen.getByText("后台核心")).toBeInTheDocument();
  expect(screen.getByText("已连接")).toBeInTheDocument();
  expect(screen.getByText("日志同步")).toBeInTheDocument();
  expect(screen.getByText("实时同步")).toBeInTheDocument();
  expect(screen.getByText("托盘待命")).toBeInTheDocument();
});
