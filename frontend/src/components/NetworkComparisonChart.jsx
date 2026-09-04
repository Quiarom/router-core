// NetworkComparisonChart was a before/after visualization that compared
// illustrative latency / download / jitter values (e.g. "86 ms -> 18 ms",
// "+287%") that were NOT produced by router-core. Those numbers were
// included in earlier dashboard drafts as a marketing-style "look how much
// better it is" widget. They are not observations, they do not come from
// the router, and they are not backed by any sanitized capture.
//
// The router-core project's core claim is that the model reasons on
// typed, provenanced router observations, never on invented numbers.
// Mixing fictitious metrics with real router telemetry in the same
// dashboard would directly contradict that claim.
//
// This component is intentionally left as a single honest placeholder.
// Anyone restoring comparison metrics must back them with sanitized
// router-core captures committed to fixtures/captured/ AND label every
// data point with its provenance.
import React from "react";

export function NetworkComparisonChart() {
  return (
    <div className="border-2 border-dashed border-neutral-800 bg-neutral-950/40 p-6 text-neutral-300 font-mono text-sm">
      <div className="flex items-center gap-2 mb-2">
        <span className="inline-block w-2 h-2 bg-neutral-600" />
        <span className="text-neutral-400 uppercase tracking-wider text-xs">
          Before/after comparison
        </span>
      </div>
      <p className="mb-2">
        This widget previously showed illustrative latency / download /
        jitter numbers ("86 ms -&gt; 18 ms", "+287%") that were NOT
        produced by router-core.
      </p>
      <p className="mb-2">
        They have been removed from the submission dashboard because they
        did not come from the physical router, had no provenance, and
        would have visually competed with the four-state capability
        matrix that the model reasons over.
      </p>
      <p className="text-neutral-500 text-xs">
        To restore comparison metrics, capture them from a real router
        via <code className="text-neutral-300">router-core-learn</code>,
        sanitize the capture, commit it under <code>fixtures/captured/</code>,
        and label every data point with its provenance.
      </p>
    </div>
  );
}
