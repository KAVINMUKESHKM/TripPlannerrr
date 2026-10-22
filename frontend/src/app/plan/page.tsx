import { Suspense } from "react";
import { PlannerClient } from "@/components/planner-client";

export default function PlanPage() {
  return <Suspense fallback={<main className="planner-shell"><p className="loading-line">Setting up your trip…</p></main>}><PlannerClient /></Suspense>;
}
