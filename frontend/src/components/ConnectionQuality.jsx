import React, { useState } from "react";
import { 
  CheckCircle2, 
  RotateCw,
  Activity
} from "lucide-react";

export function ConnectionQuality() {
  const [activeMetric, setActiveMetric] = useState("download");
  const [isTesting, setIsTesting] = useState(false);
  const [hoverIndex, setHoverIndex] = useState(null);

  const timelineData = {
    latency: {
      label: "Latencia / Ping",
      unit: "ms",
      beforeAvg: "86 ms",
      afterAvg: "18 ms",
      diff: "-79% de demora",
      pointsBefore: [82, 115, 94, 130, 88, 145, 92, 120, 78, 110],
      pointsAfter:  [19, 18, 17, 18, 19, 17, 18, 18, 19, 17],
      yMax: 160,
      description: "Menor es mejor. Bajar de 86ms a 18ms elimina las demoras por completo en llamadas y navegación."
    },
    download: {
      label: "Velocidad de Descarga",
      unit: "Mbps",
      beforeAvg: "24.5 Mbps",
      afterAvg: "94.8 Mbps",
      diff: "+287% de velocidad",
      pointsBefore: [22, 19, 28, 14, 25, 20, 31, 18, 23, 26],
      pointsAfter:  [92, 95, 94, 96, 93, 97, 95, 94, 98, 96],
      yMax: 110,
      description: "Mayor es mejor. Con 95 Mbps puedes reproducir contenido 4K en varios dispositivos a la vez sin interrupciones."
    },
    jitter: {
      label: "Estabilidad / Jitter",
      unit: "ms",
      beforeAvg: "24 ms",
      afterAvg: "2 ms",
      diff: "-91% de variación",
      pointsBefore: [25, 34, 18, 40, 22, 38, 19, 29, 32, 21],
      pointsAfter:  [2, 2, 3, 1, 2, 2, 2, 3, 2, 2],
      yMax: 50,
      description: "Menor es mejor. Indica si la señal sufre altibajos; con 2ms la transmisión de datos es totalmente estable."
    }
  };

  const current = timelineData[activeMetric];

  // Larger graph dimensions for prominent display
  const width = 800;
  const height = 280;
  const paddingX = 40;
  const paddingY = 32;

  const getCoordinates = (points, yMax) => {
    const step = (width - paddingX * 2) / (points.length - 1);
    return points.map((val, idx) => {
      const x = paddingX + idx * step;
      const y = height - paddingY - (val / yMax) * (height - paddingY * 2);
      return { x, y, val };
    });
  };

  const beforeCoords = getCoordinates(current.pointsBefore, current.yMax);
  const afterCoords = getCoordinates(current.pointsAfter, current.yMax);

  const makePath = (coords) => {
    return coords.reduce((acc, pt, i) => `${acc} ${i === 0 ? "M" : "L"} ${pt.x},${pt.y}`, "");
  };

  const beforeSvgPath = makePath(beforeCoords);
  const afterSvgPath = makePath(afterCoords);

  const triggerSpeedTest = () => {
    setIsTesting(true);
    setTimeout(() => {
      setIsTesting(false);
    }, 1000);
  };

  return (
    <div className="w-full border-2 border-neutral-800 bg-neutral-950 p-4 sm:p-5 shadow-sm space-y-5 text-white">
      {/* Title bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b-2 border-neutral-800 pb-5 pt-1">
        <div>
          <h2 className="text-xl sm:text-2xl font-black uppercase tracking-tight text-white font-sans">
            Comparativa de Conexión
          </h2>
          <p className="text-xs text-neutral-400 font-mono mt-0.5">
            Diagnóstico continuo de respuesta, velocidad y estabilidad en el router
          </p>
        </div>

        <button
          onClick={triggerSpeedTest}
          disabled={isTesting}
          className="px-4 py-2 bg-neutral-900 hover:bg-neutral-800 border-2 border-neutral-700 text-xs font-mono font-bold uppercase tracking-wider text-white flex items-center gap-2 transition-colors cursor-pointer self-start sm:self-auto disabled:opacity-50"
        >
          <RotateCw className={`h-3.5 w-3.5 ${isTesting ? "animate-spin text-primary" : ""}`} />
          <span>{isTesting ? "Midiendo..." : "Repetir Test"}</span>
        </button>
      </div>

      {/* 4 Emphasized Metric Cards: Centered numbers, emphasized labels, bigger percentages */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 font-mono">
        {/* Latency */}
        <button
          type="button"
          onClick={() => setActiveMetric("latency")}
          className={`flex flex-col items-center justify-center text-center p-5 border-2 transition-all cursor-pointer ${
            activeMetric === "latency"
              ? "border-primary bg-primary/10 shadow-sm"
              : "border-neutral-800 bg-neutral-900 hover:border-neutral-600"
          }`}
        >
          <span className="text-xs sm:text-sm font-bold uppercase tracking-wider text-neutral-400">
            LATENCIA / PING
          </span>
          <div className="text-2xl sm:text-3xl lg:text-4xl font-black text-white mt-1.5 tracking-tight">
            18 <span className="text-sm font-bold text-neutral-400">ms</span>
          </div>
          <span className="text-xs sm:text-sm font-bold text-primary mt-1.5 block">
            -79% de demora
          </span>
        </button>

        {/* Download */}
        <button
          type="button"
          onClick={() => setActiveMetric("download")}
          className={`flex flex-col items-center justify-center text-center p-5 border-2 transition-all cursor-pointer ${
            activeMetric === "download"
              ? "border-primary bg-primary/10 shadow-sm"
              : "border-neutral-800 bg-neutral-900 hover:border-neutral-600"
          }`}
        >
          <span className="text-xs sm:text-sm font-bold uppercase tracking-wider text-neutral-400">
            DESCARGA
          </span>
          <div className="text-2xl sm:text-3xl lg:text-4xl font-black text-white mt-1.5 tracking-tight">
            94.8 <span className="text-sm font-bold text-neutral-400">Mbps</span>
          </div>
          <span className="text-xs sm:text-sm font-bold text-primary mt-1.5 block">
            +287% de velocidad
          </span>
        </button>

        {/* Jitter */}
        <button
          type="button"
          onClick={() => setActiveMetric("jitter")}
          className={`flex flex-col items-center justify-center text-center p-5 border-2 transition-all cursor-pointer ${
            activeMetric === "jitter"
              ? "border-primary bg-primary/10 shadow-sm"
              : "border-neutral-800 bg-neutral-900 hover:border-neutral-600"
          }`}
        >
          <span className="text-xs sm:text-sm font-bold uppercase tracking-wider text-neutral-400">
            ESTABILIDAD (JITTER)
          </span>
          <div className="text-2xl sm:text-3xl lg:text-4xl font-black text-white mt-1.5 tracking-tight">
            2 <span className="text-sm font-bold text-neutral-400">ms</span>
          </div>
          <span className="text-xs sm:text-sm font-bold text-primary mt-1.5 block">
            -91% de variación
          </span>
        </button>

        {/* Packet Loss */}
        <div className="flex flex-col items-center justify-center text-center p-5 border-2 border-neutral-800 bg-neutral-900">
          <span className="text-xs sm:text-sm font-bold uppercase tracking-wider text-neutral-400">
            PÉRDIDA DE DATOS
          </span>
          <div className="text-2xl sm:text-3xl lg:text-4xl font-black text-white mt-1.5 tracking-tight">
            0.0 <span className="text-sm font-bold text-neutral-400">%</span>
          </div>
          <span className="text-xs sm:text-sm font-bold text-emerald-400 mt-1.5 block">
            0% pérdida • LIMPIO
          </span>
        </div>
      </div>

      {/* SVG Interactive Chart Box (Enlarged graph & larger title) */}
      <div className="border-2 border-neutral-800 bg-black p-4 sm:p-5 relative space-y-3">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-3 border-b border-neutral-800">
          {/* Bigger Title for Curve */}
          <div className="flex items-center gap-2">
            <Activity className="h-4 w-4 text-primary shrink-0" />
            <span className="text-base sm:text-lg font-bold font-mono text-white tracking-tight uppercase">
              CURVA DE {current.label} ({current.unit})
            </span>
          </div>

          <div className="flex items-center gap-4 text-xs font-mono">
            <div className="flex items-center gap-1.5">
              <span className="w-3 h-0.5 bg-rose-500" />
              <span className="text-neutral-400">Antes</span>
            </div>
            <div className="flex items-center gap-1.5">
              <span className="w-3 h-0.5 bg-primary" />
              <span className="text-primary font-bold">Después (Optimizado)</span>
            </div>
          </div>
        </div>

        {/* Larger Taller SVG Chart */}
        <div className="w-full overflow-x-auto py-1">
          <svg
            viewBox={`0 0 ${width} ${height}`}
            className="w-full h-64 sm:h-72 lg:h-80 select-none"
          >
            {/* Grid lines */}
            {[0, 0.25, 0.5, 0.75, 1].map((ratio, idx) => {
              const y = height - paddingY - ratio * (height - paddingY * 2);
              const labelVal = Math.round(ratio * current.yMax);
              return (
                <g key={idx}>
                  <line
                    x1={paddingX}
                    y1={y}
                    x2={width - paddingX}
                    y2={y}
                    stroke="#262626"
                    strokeDasharray="4 4"
                    strokeWidth="1"
                  />
                  <text
                    x={paddingX - 8}
                    y={y + 3}
                    fill="#737373"
                    fontSize="11"
                    fontFamily="monospace"
                    textAnchor="end"
                  >
                    {labelVal}
                  </text>
                </g>
              );
            })}

            {/* Time labels */}
            {beforeCoords.map((pt, idx) => (
              <text
                key={idx}
                x={pt.x}
                y={height - 10}
                fill="#737373"
                fontSize="11"
                fontFamily="monospace"
                textAnchor="middle"
              >
                {idx * 5}s
              </text>
            ))}

            {/* Before Path (Red dashed) */}
            <path
              d={beforeSvgPath}
              fill="none"
              stroke="#EF4444"
              strokeWidth="2"
              strokeDasharray="4 3"
            />

            {/* After Path (Purple solid) */}
            <path
              d={afterSvgPath}
              fill="none"
              stroke="#8B5CF6"
              strokeWidth="3.5"
            />

            {/* Before Dots (Red) */}
            {beforeCoords.map((pt, idx) => (
              <circle
                key={`b-${idx}`}
                cx={pt.x}
                cy={pt.y}
                r="3"
                fill="#EF4444"
              />
            ))}

            {/* After Dots (Purple) */}
            {afterCoords.map((pt, idx) => (
              <circle
                key={`a-${idx}`}
                cx={pt.x}
                cy={pt.y}
                r={hoverIndex === idx ? 6 : 4}
                fill="#8B5CF6"
                stroke="#000000"
                strokeWidth="1.5"
                onMouseEnter={() => setHoverIndex(idx)}
                onMouseLeave={() => setHoverIndex(null)}
                className="cursor-pointer transition-all"
              />
            ))}
          </svg>
        </div>

        {/* Minimal Explanation without bulky background (Removed Prueba #2) */}
        <div className="border-l-2 border-primary pl-3 py-1 flex items-center justify-between text-xs font-sans">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-primary shrink-0" />
            <span className="text-neutral-300">
              <strong className="text-white">Diagnóstico: </strong>
              {current.description}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

export default ConnectionQuality;
