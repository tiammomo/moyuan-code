import { Suspense } from "react";
import { ConsoleWorkbench } from "@/components/console-workbench";
import { getConsoleSnapshot } from "@/lib/api";
import { demoSnapshot } from "@/lib/demo-data";

export default function ConsolePage() {
  return (
    <Suspense fallback={<ConsoleWorkbench snapshot={demoSnapshot} />}>
      <ConsoleData />
    </Suspense>
  );
}

async function ConsoleData() {
  const snapshot = await getConsoleSnapshot();

  return <ConsoleWorkbench snapshot={snapshot} />;
}
