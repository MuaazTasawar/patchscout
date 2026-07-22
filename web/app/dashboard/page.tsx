import RealtimeDashboard from "@/components/RealtimeDashboard";

export default function DashboardPage() {
  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-semibold font-mono">Review Queue</h1>
        <p className="text-text-secondary text-sm mt-1">
          Every finding lands here before anything is posted publicly. Approve,
          reject, or edit the draft text — nothing goes live automatically.
        </p>
      </div>
      <RealtimeDashboard />
    </div>
  );
}