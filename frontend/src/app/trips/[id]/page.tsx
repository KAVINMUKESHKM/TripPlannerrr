import { PlannerClient } from "@/components/planner-client";

export default async function TripDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <PlannerClient tripId={id} />;
}
