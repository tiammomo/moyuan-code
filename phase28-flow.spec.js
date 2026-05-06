const { expect, test } = require("playwright/test");

const baseURL = "http://127.0.0.1:3000?project=moyuan-code&refresh=phase28-browser-flow";

test("phase 28 requirement and issue flow", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 980 });
  await page.goto(baseURL, { waitUntil: "networkidle" });
  await expect(page.getByRole("heading", { name: "moyuan-code" })).toBeVisible();
  await page.screenshot({ path: "/tmp/phase28-01-project.png", fullPage: true });

  await page.getByRole("button", { name: "需求与 Issue" }).click();
  await expect(page.getByRole("strong").filter({ hasText: /Phase 28/ }).first()).toBeVisible();
  await expect(page.locator(".requirementRecord").first()).toContainText("Phase 28");
  await expect(page.getByText("0 已完成 / 5 总数")).toBeVisible();
  await expect(page.locator(".requirementLedgerPanel")).not.toContainText("�");
  await page.screenshot({ path: "/tmp/phase28-02-requirements.png", fullPage: true });

  await page.getByRole("tab", { name: "Issue Graph" }).click();
  await expect(page.getByText("待执行 3 节点 / 2 依赖")).toBeVisible();
  await expect(page.getByRole("button", { name: /查看 Issue requirement-contract/ })).toBeVisible();
  await expect(page.getByRole("button", { name: /查看 Issue backend-implementation/ })).toBeVisible();
  await expect(page.getByRole("button", { name: /查看 Issue quality-review/ })).toBeVisible();
  await page.screenshot({ path: "/tmp/phase28-03-issue-graph.png", fullPage: true });

  await page.getByRole("button", { name: /查看 Issue backend-implementation/ }).click();
  await expect(page.getByRole("heading", { name: "backend-implementation" })).toBeVisible();
  await expect(page.locator(".dependencyList")).toContainText("requirement-contract");
  await page.screenshot({ path: "/tmp/phase28-04-issue-inspector.png", fullPage: true });

  await page.getByRole("tab", { name: "批量执行" }).click();
  await page.getByRole("button", { name: "创建计划" }).click();
  await expect(page.getByRole("heading", { name: "创建批量计划" })).toBeVisible();
  const epicValue = await page.getByLabel("Epic ID").inputValue();
  expect(epicValue).toContain("phase-28");
  await page.screenshot({ path: "/tmp/phase28-05-batch-modal.png", fullPage: true });
  await page.getByRole("button", { name: "创建计划" }).last().click();
  await expect(page.getByText("批量计划就绪").first()).toBeVisible({ timeout: 10000 });
  await page.screenshot({ path: "/tmp/phase28-06-batch-created.png", fullPage: true });

  const dryRunResponse = page.waitForResponse(
    (response) =>
      response.request().method() === "POST" &&
      response.url().includes("/api/projects/moyuan-code/batches/") &&
      response.url().endsWith("/run"),
  );
  await page.getByRole("button", { name: "试运行" }).first().click();
  expect((await dryRunResponse).ok()).toBeTruthy();
  await expect(page.locator(".actionMessage.completed").filter({ hasText: "dry-run 已完成" }).first()).toBeVisible({ timeout: 10000 });
  await expect(page.locator(".actionMessage.running").filter({ hasText: "正在创建 dry-run 运行" })).toHaveCount(0);
  await page.screenshot({ path: "/tmp/phase28-07-batch-dry-run.png", fullPage: true });
});
