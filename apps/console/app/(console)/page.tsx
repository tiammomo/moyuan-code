import { Suspense } from "react";
import { ConsoleWorkbench } from "@/components/console-workbench";
import { getConsoleSnapshot } from "@/lib/api";

type ConsolePageSearchParams = Promise<Record<string, string | string[] | undefined>>;

export default function ConsolePage({ searchParams }: { searchParams?: ConsolePageSearchParams }) {
  return (
    <Suspense fallback={<ConsoleLoading />}>
      <ConsoleData searchParams={searchParams} />
    </Suspense>
  );
}

async function ConsoleData({ searchParams }: { searchParams?: ConsolePageSearchParams }) {
  const params = searchParams ? await searchParams : {};
  const snapshot = await getConsoleSnapshot(readSearchParam(params.project));

  return <ConsoleWorkbench snapshot={snapshot} />;
}

function readSearchParam(value: string | string[] | undefined) {
  return Array.isArray(value) ? value[0] : value;
}

function ConsoleLoading() {
  return <main className="consoleLoading">正在加载控制台...</main>;
}
