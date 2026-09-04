import React, { useEffect, useState } from "react";
import {
  AlertCircle,
  Bot,
  CheckCircle2,
  Eye,
  EyeOff,
  LoaderCircle,
  LockKeyhole,
  Radio,
  Router
} from "lucide-react";

import { connectRouter, detectGateway } from "@/lib/desktop";

const SETUP_TEXT = {
  title: "Conecta tu router",
  subtitle: "La aplicación detecta el gateway y conserva las credenciales únicamente en memoria.",
  host: "Dirección del router",
  username: "Usuario administrador",
  password: "Contraseña del router",
  apiKey: "AI provider API key (opcional)",
  apiHint: "Sin clave, el asistente funciona en modo determinista con datos reales.",
  security: "La contraseña se entrega directamente al servicio local y nunca se guarda en disco.",
  connect: "Conectar y leer datos",
  connecting: "Validando router...",
  supportTitle: "Compatibilidad universal",
  supportBody: "Diseñado para conectarse con cualquier router mediante interfaz HTTP local. Detecta automáticamente la identidad y capacidades del dispositivo."
};

/**
 * Muestra el asistente inicial y entrega una sesión validada al componente raíz.
 *
 * @param {{onConnected: (connection: object) => void}} props Propiedades del flujo inicial.
 * @returns {React.JSX.Element} Formulario local de conexión al router.
 */
export function RouterSetup({ onConnected }) {
  const [host, setHost] = useState("");
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [aiApiKey, setAiApiKey] = useState("");
  const [isPasswordVisible, setIsPasswordVisible] = useState(false);
  const [isDetecting, setIsDetecting] = useState(true);
  const [isConnecting, setIsConnecting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    let isCancelled = false;
    detectGateway()
      .then((gateway) => {
        if (!isCancelled) setHost(gateway);
      })
      .catch((error) => {
        if (!isCancelled) {
          setErrorMessage(String(error));
        }
      })
      .finally(() => {
        if (!isCancelled) setIsDetecting(false);
      });
    return () => {
      isCancelled = true;
    };
  }, []);

  const handleSubmit = async (event) => {
    event.preventDefault();
    setErrorMessage("");
    setIsConnecting(true);

    try {
      const connection = await connectRouter({
        host: host.trim(),
        username: username.trim(),
        password,
        aiApiKey: aiApiKey.trim() || undefined
      });
      setPassword("");
      setAiApiKey("");
      onConnected(connection);
    } catch (error) {
      setErrorMessage(String(error));
    } finally {
      setIsConnecting(false);
    }
  };

  return (
    <main className="min-h-screen bg-black text-white grid place-items-center px-4 py-10 font-sans">
      <div className="w-full max-w-3xl border-2 border-neutral-800 bg-neutral-950">
        <header className="border-b-2 border-neutral-800 p-6 sm:p-8 space-y-3">
          <div className="flex items-center gap-3">
            <div className="grid h-11 w-11 place-items-center border-2 border-primary bg-primary/10">
              <Router className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="font-mono text-xs font-bold uppercase tracking-[0.2em] text-primary">
                router-core desktop
              </p>
              <h1 className="text-2xl sm:text-3xl font-black uppercase tracking-tight">
                {SETUP_TEXT.title}
              </h1>
            </div>
          </div>
          <p className="text-sm text-neutral-400 max-w-2xl">{SETUP_TEXT.subtitle}</p>
        </header>

        <form onSubmit={handleSubmit} className="p-6 sm:p-8 space-y-5">
          {errorMessage && (
            <div className="flex items-start gap-3 border-2 border-rose-500/40 bg-rose-500/10 p-4 text-sm text-rose-200">
              <AlertCircle className="h-5 w-5 shrink-0 text-rose-400" />
              <span>{errorMessage}</span>
            </div>
          )}

          <div className="grid gap-5 sm:grid-cols-2">
            <label className="space-y-2">
              <span className="flex items-center gap-2 font-mono text-xs font-bold uppercase text-neutral-300">
                <Radio className="h-4 w-4 text-primary" />
                {SETUP_TEXT.host}
              </span>
              <div className="relative">
                <input
                  required
                  value={host}
                  onChange={(event) => setHost(event.target.value)}
                  placeholder="192.168.1.1"
                  className="w-full border-2 border-neutral-700 bg-black px-3 py-3 font-mono text-sm text-white outline-none focus:border-primary"
                />
                {isDetecting && (
                  <LoaderCircle className="absolute right-3 top-3.5 h-4 w-4 animate-spin text-primary" />
                )}
              </div>
            </label>

            <label className="space-y-2">
              <span className="font-mono text-xs font-bold uppercase text-neutral-300">
                {SETUP_TEXT.username}
              </span>
              <input
                required
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                autoComplete="username"
                className="w-full border-2 border-neutral-700 bg-black px-3 py-3 font-mono text-sm text-white outline-none focus:border-primary"
              />
            </label>
          </div>

          <label className="space-y-2 block">
            <span className="flex items-center gap-2 font-mono text-xs font-bold uppercase text-neutral-300">
              <LockKeyhole className="h-4 w-4 text-primary" />
              {SETUP_TEXT.password}
            </span>
            <div className="relative">
              <input
                required
                type={isPasswordVisible ? "text" : "password"}
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                autoComplete="current-password"
                className="w-full border-2 border-neutral-700 bg-black px-3 py-3 pr-12 font-mono text-sm text-white outline-none focus:border-primary"
              />
              <button
                type="button"
                onClick={() => setIsPasswordVisible((value) => !value)}
                className="absolute right-2 top-2 grid h-8 w-8 place-items-center text-neutral-400 hover:text-white"
                aria-label={isPasswordVisible ? "Ocultar contraseña" : "Mostrar contraseña"}
              >
                {isPasswordVisible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </label>

          <label className="space-y-2 block">
            <span className="flex items-center gap-2 font-mono text-xs font-bold uppercase text-neutral-300">
              <Bot className="h-4 w-4 text-primary" />
              {SETUP_TEXT.apiKey}
            </span>
            <input
              type="password"
              value={aiApiKey}
              onChange={(event) => setAiApiKey(event.target.value)}
              autoComplete="off"
              placeholder="sk-or-v1-..."
              className="w-full border-2 border-neutral-700 bg-black px-3 py-3 font-mono text-sm text-white outline-none focus:border-primary"
            />
            <span className="block text-xs text-neutral-500">{SETUP_TEXT.apiHint}</span>
          </label>

          <div className="grid gap-3 sm:grid-cols-2">
            <div className="flex items-start gap-2 border border-neutral-800 bg-black p-3 text-xs text-neutral-400">
              <CheckCircle2 className="h-4 w-4 shrink-0 text-emerald-400" />
              <span>{SETUP_TEXT.security}</span>
            </div>
            <div className="flex items-start gap-2 border border-amber-500/30 bg-amber-500/5 p-3 text-xs text-amber-100/80">
              <AlertCircle className="h-4 w-4 shrink-0 text-amber-400" />
              <span>
                <strong className="block text-amber-300">{SETUP_TEXT.supportTitle}</strong>
                {SETUP_TEXT.supportBody}
              </span>
            </div>
          </div>

          <button
            type="submit"
            disabled={isConnecting || isDetecting}
            className="flex w-full items-center justify-center gap-2 border-2 border-primary bg-primary px-5 py-3 font-mono text-sm font-black uppercase text-white transition-colors hover:bg-primary-hover disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isConnecting ? (
              <LoaderCircle className="h-4 w-4 animate-spin" />
            ) : (
              <Radio className="h-4 w-4" />
            )}
            {isConnecting ? SETUP_TEXT.connecting : SETUP_TEXT.connect}
          </button>
        </form>
      </div>
    </main>
  );
}
