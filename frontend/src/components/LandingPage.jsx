import React, { useState } from "react";
import { 
  ShieldCheck, 
  Cpu, 
  Lock, 
  Radio, 
  ArrowRight, 
  CheckCircle2, 
  Code2, 
  Sparkles,
  Server,
  Eye,
  Copy,
  Check
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { mockDevice, mockStatus, mockClients, mockCapabilities, mockSecurity } from "@/data/mockData";

export function LandingPage({ onOpenDashboard }) {
  const [selectedEndpoint, setSelectedEndpoint] = useState("/v0/device");
  const [copied, setCopied] = useState(false);

  const getEndpointData = (endpoint) => {
    switch (endpoint) {
      case "/v0/device":
        return mockDevice;
      case "/v0/status":
        return mockStatus;
      case "/v0/clients":
        return mockClients;
      case "/v0/capabilities":
        return mockCapabilities;
      case "/v0/security/dmz":
        return mockSecurity.dmz;
      default:
        return mockDevice;
    }
  };

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="space-y-24 py-8 pb-20">
      {/* Hero Section */}
      <section className="relative overflow-hidden pt-6 pb-12 sm:pt-12">
        <div className="absolute inset-0 -z-10 bg-neutral-900" />
        
        <div className="mx-auto max-w-5xl text-center px-4 sm:px-6">
          <div className="inline-flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-400 mb-8">
            <Sparkles className="h-3.5 w-3.5" />
            <span>Submission for GMI Cloud × MiniMax Week (Track: Reasoning)</span>
          </div>

          <h1 className="text-4xl font-extrabold tracking-tight text-white sm:text-6xl font-sans">
            A Safe, Read-Only AI Layer for <br className="hidden sm:block" />
            <span className="bg-gradient-to-r from-emerald-400 via-teal-300 to-cyan-400 bg-clip-text text-transparent">
              Legacy Consumer Routers
            </span>
          </h1>

          <p className="mt-6 text-lg text-slate-300 max-w-3xl mx-auto leading-relaxed">
            Turns 2013 consumer routers into a typed HTTP API on loopback that a{" "}
            <strong className="text-emerald-400 font-semibold">MiniMax M3 reasoning layer</strong> can safely observe.
            Mutations are unrepresentable. Zero writes. RFC1918 strictly enforced.
          </p>

          <div className="mt-10 flex flex-wrap items-center justify-center gap-4">
            <Button 
              size="lg" 
              onClick={onOpenDashboard}
              className="gap-2 bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-semibold shadow-lg shadow-emerald-950/40"
            >
              Open Live Dashboard
              <ArrowRight className="h-4 w-4" />
            </Button>
            <a 
              href="#api-explorer" 
              className="inline-flex items-center gap-2 rounded-lg border border-slate-700 bg-slate-900/80 px-6 py-2.5 text-sm font-medium text-slate-200 hover:bg-slate-800 transition-colors"
            >
              <Code2 className="h-4 w-4 text-emerald-400" />
              Explore API Endpoints
            </a>
          </div>

          {/* Device & Architecture pill row */}
          <div className="mt-12 flex flex-wrap items-center justify-center gap-3 text-xs text-slate-400">
            <div className="flex items-center gap-1.5 px-3 py-1 rounded-md border border-slate-800 bg-slate-900/60 font-mono">
              <Radio className="h-3 w-3 text-emerald-400" />
              Target: TP-Link TL-WR841N v8.4
            </div>
            <div className="flex items-center gap-1.5 px-3 py-1 rounded-md border border-slate-800 bg-slate-900/60 font-mono">
              <Cpu className="h-3 w-3 text-cyan-400" />
              Model: MiniMax M3 (1M context)
            </div>
            <div className="flex items-center gap-1.5 px-3 py-1 rounded-md border border-slate-800 bg-slate-900/60 font-mono">
              <Lock className="h-3 w-3 text-amber-400" />
              Transport: GET only • 2 MiB cap
            </div>
          </div>
        </div>
      </section>

      {/* Safety Invariants Section */}
      <section className="mx-auto max-w-6xl px-4 sm:px-6">
        <div className="text-center mb-12">
          <h2 className="text-2xl font-bold tracking-tight text-white sm:text-3xl">
            Invariants Hardened by Design
          </h2>
          <p className="mt-2 text-sm text-slate-400 max-w-xl mx-auto">
            Traditional tools guess protocols or risk rebooting home networks. router-core enforces non-negotiable safety rules.
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
          <Card className="border-slate-800 bg-slate-900/40">
            <CardHeader>
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400 mb-2">
                <Eye className="h-5 w-5" />
              </div>
              <CardTitle>GET-Only Invariant</CardTitle>
              <CardDescription>
                The transport client rejects any POST, PUT or DELETE. The mutation path (<code className="text-emerald-400">CapMutate</code>) does not exist in the code.
              </CardDescription>
            </CardHeader>
          </Card>

          <Card className="border-slate-800 bg-slate-900/40">
            <CardHeader>
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-cyan-500/10 text-cyan-400 mb-2">
                <Lock className="h-5 w-5" />
              </div>
              <CardTitle>RFC1918 Loopback Only</CardTitle>
              <CardDescription>
                Refuses public internet IPs, domain names, and cross-host redirects. The runtime only talks directly to your private router gateway.
              </CardDescription>
            </CardHeader>
          </Card>

          <Card className="border-slate-800 bg-slate-900/40">
            <CardHeader>
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-500/10 text-amber-400 mb-2">
                <ShieldCheck className="h-5 w-5" />
              </div>
              <CardTitle>First-Class Unknown</CardTitle>
              <CardDescription>
                Absence of data is never turned into <code className="text-amber-300">false</code>. States explicitly reflect: <em>verified</em>, <em>absent</em>, <em>unverified</em>, or <em>unavailable</em>.
              </CardDescription>
            </CardHeader>
          </Card>

          <Card className="border-slate-800 bg-slate-900/40">
            <CardHeader>
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-500/10 text-purple-400 mb-2">
                <Server className="h-5 w-5" />
              </div>
              <CardTitle>2 MiB Response Cap</CardTitle>
              <CardDescription>
                Guards local process memory against unbounded firmware memory dumps or infinite streaming loops from legacy routers.
              </CardDescription>
            </CardHeader>
          </Card>
        </div>
      </section>

      {/* Interactive API Explorer */}
      <section id="api-explorer" className="mx-auto max-w-6xl px-4 sm:px-6">
        <div className="flex flex-col md:flex-row md:items-end justify-between mb-8 gap-4">
          <div>
            <div className="inline-flex items-center gap-1.5 text-xs font-mono text-emerald-400 mb-2">
              <Code2 className="h-4 w-4" />
              <span>STABLE HTTP SURFACE</span>
            </div>
            <h2 className="text-2xl font-bold tracking-tight text-white sm:text-3xl">
              Contract-First API Explorer
            </h2>
            <p className="mt-1 text-sm text-slate-400">
              Defined in <code className="text-slate-300">docs/FRONTEND_CONTRACT.md</code> and consumed by both this UI and MiniMax M3.
            </p>
          </div>

          <div className="flex flex-wrap gap-2">
            {[
              "/v0/device",
              "/v0/status",
              "/v0/clients",
              "/v0/capabilities",
              "/v0/security/dmz"
            ].map((endpoint) => (
              <button
                key={endpoint}
                onClick={() => setSelectedEndpoint(endpoint)}
                className={`px-3 py-1.5 rounded-lg text-xs font-mono transition-colors ${
                  selectedEndpoint === endpoint
                    ? "bg-emerald-500/20 text-emerald-300 border border-emerald-500/40"
                    : "bg-slate-900 text-slate-400 border border-slate-800 hover:text-slate-200"
                }`}
              >
                {endpoint}
              </button>
            ))}
          </div>
        </div>

        <div className="rounded-xl border border-slate-800 bg-slate-950 overflow-hidden shadow-2xl">
          <div className="flex items-center justify-between border-b border-slate-850 px-4 py-3 bg-slate-900/60">
            <div className="flex items-center gap-3">
              <span className="rounded bg-emerald-500/20 px-2 py-0.5 text-xs font-mono font-semibold text-emerald-400">
                GET
              </span>
              <span className="text-xs font-mono text-slate-200">
                http://127.0.0.1:8484{selectedEndpoint}
              </span>
            </div>
            <button
              onClick={() => copyToClipboard(JSON.stringify(getEndpointData(selectedEndpoint), null, 2))}
              className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-slate-200 font-mono transition-colors"
            >
              {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
              <span>{copied ? "Copied" : "Copy JSON"}</span>
            </button>
          </div>
          <div className="p-4 overflow-x-auto max-h-96">
            <pre className="font-mono text-xs text-slate-300 leading-relaxed">
              {JSON.stringify(getEndpointData(selectedEndpoint), null, 2)}
            </pre>
          </div>
        </div>
      </section>

      {/* MiniMax Reasoning Workflow */}
      <section className="mx-auto max-w-6xl px-4 sm:px-6">
        <div className="rounded-2xl border border-slate-800/80 bg-gradient-to-b from-slate-900/60 to-slate-950 p-8 sm:p-12 shadow-xl">
          <div className="max-w-3xl">
            <div className="inline-flex items-center gap-2 rounded-md bg-purple-500/10 px-2.5 py-1 text-xs font-medium text-purple-400 mb-4 border border-purple-500/20">
              <Cpu className="h-3.5 w-3.5" />
              Reasoning Engine (MiniMax M3)
            </div>
            <h2 className="text-2xl font-bold tracking-tight text-white sm:text-3xl">
              From Raw JavaScript Dashboards to Natural Language Answers
            </h2>
            <p className="mt-3 text-sm text-slate-300 leading-relaxed">
              Instead of an operator deciphering 2013 web forms, a question like{" "}
              <span className="text-emerald-400 font-mono font-medium">"¿Está expuesta mi red?"</span> triggers a deterministic, read-only sequence of tool calls:
            </p>

            <div className="mt-6 space-y-3 font-mono text-xs">
              <div className="flex items-center gap-3 rounded-lg border border-slate-800 bg-slate-950/80 p-3 text-slate-300">
                <CheckCircle2 className="h-4 w-4 text-emerald-400 shrink-0" />
                <span>1. Agent queries <strong className="text-white">/v0/device</strong> &amp; <strong className="text-white">/v0/capabilities</strong> to establish ground truth.</span>
              </div>
              <div className="flex items-center gap-3 rounded-lg border border-slate-800 bg-slate-950/80 p-3 text-slate-300">
                <CheckCircle2 className="h-4 w-4 text-emerald-400 shrink-0" />
                <span>2. Checks <strong className="text-white">/v0/security/dmz</strong> and <strong className="text-white">/v0/security/forwarding</strong> for exposed hosts.</span>
              </div>
              <div className="flex items-center gap-3 rounded-lg border border-slate-800 bg-slate-950/80 p-3 text-slate-300">
                <CheckCircle2 className="h-4 w-4 text-emerald-400 shrink-0" />
                <span>3. MiniMax synthesizes findings with zero mutation risk and presents clear, human recommendations.</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Quickstart Banner */}
      <section className="mx-auto max-w-4xl px-4 text-center">
        <h3 className="text-xl font-bold text-white mb-3">Ready to inspect your router?</h3>
        <p className="text-sm text-slate-400 mb-6">
          Build the Go binary and start the HTTP observation server on loopback in seconds.
        </p>
        <div className="inline-flex items-center justify-between rounded-lg border border-slate-800 bg-slate-950 px-4 py-3 font-mono text-xs text-slate-300 max-w-xl w-full">
          <span>./bin/router-core serve --host 192.168.1.1 --addr 127.0.0.1:8484</span>
          <Button size="sm" onClick={onOpenDashboard} className="h-7 text-xs ml-3">
            Open Dashboard
          </Button>
        </div>
      </section>
    </div>
  );
}
