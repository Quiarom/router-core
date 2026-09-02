import React, { useState } from "react";
import { 
  CheckCircle2, 
  RotateCw
} from "lucide-react";

export function NetworkComparisonChart() {
  const [activeMetric, setActiveMetric] = useState("latency");
  const [isTesting, setIsTesting] = useState(false);
  const [testCount, setTestCount] = useState(1);
  const [hoverIndex, setHoverIndex] = useState(null);

  const timelineData = {
    latency: {
      label: "Latencia / Ping",
      unit: "ms",
      beforeAvg: "86 ms",
      afterAvg: "18 ms",
      diff: "-79%",
      pointsBefore: [82, 115, 94, 130, 88, 145, 92, 120, 78, 110],
      pointsAfter:  [19, 18, 17, 18, 19, 17, 18, 18, 19, 17],
      yMax: 160,
      description: "Menor es mejor. Mide el tiempo de respuesta. Bajar de 86ms a 18ms elimina las demoras por completo."
    },
    download: {
      label: "Velocidad de Descarga",
      unit: "Mbps",
      beforeAvg: "24.5 Mbps",
      afterAvg: "94.8 Mbps",
      diff: "+287%",
      pointsBefore: [22, 19, 28, 14, 25, 20, 31, 18, 23, 26],
      pointsAfter:  [92, 95, 94, 96, 93, 97, 95, 94, 98, 96],
      yMax: 110,
      description: "Mayor es mejor. La velocidad a la que cargan videos y páginas web. Casi 4 veces más veloz."
    },
    jitter: {
      label: "Estabilidad / Jitter",
      unit: "ms",
      beforeAvg: "24 ms",
      afterAvg: "2 ms",
      diff: "-91%",
      pointsBefore: [25, 34, 18, 40, 22, 38, 19, 29, 32, 21],
      pointsAfter:  [2, 2, 3, 1, 2, 2, 2, 3, 2, 2],
      yMax: 50,
      description: "Menor es mejor. Indica si la señal sufre altibajos. Con 2ms la llamada no se congela."
    }
  };

  const current = timelineData[activeMetric];

  const width = 800;
  const height = 220;
  const paddingX = 40;
  const paddingY = 30;

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
      setTestCount((c) => c + 1);
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

      {/* Minimal Telemetry Columns (Enlarged & Clear) */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-6 pt-1 font-mono">
        {/* Latency */}
        <div 
          onClick={() => setActiveMetric("latency")}
          className={`border-l-2 pl-3 space-y-1 cursor-pointer transition-colors ${
            activeMetric === "latency" ? "border-primary" : "border-neutral-800 hover:border-neutral-600"
          }`}
        >
          <span className="text-neutral-500 uppercase text-xs block font-medium">LATENCIA / PING:</span>
          <div className="text-2xl sm:text-3xl font-black text-white tracking-tight">
            18 <span className="text-sm font-bold text-neutral-400">ms</span>
          </div>
          <span className="text-xs text-primary font-bold block">-79% de demora</span>
        </div>

        {/* Download */}
        <div 
          onClick={() => setActiveMetric("download")}
          className={`border-l-2 pl-3 space-y-1 cursor-pointer transition-colors ${
            activeMetric === "download" ? "border-primary" : "border-neutral-800 hover:border-neutral-600"
          }`}
        >
          <span className="text-neutral-500 uppercase text-xs block font-medium">DESCARGA:</span>
          <div className="text-2xl sm:text-3xl font-black text-white tracking-tight">
            94.8 <span className="text-sm font-bold text-neutral-400">Mbps</span>
          </div>
          <span className="text-xs text-primary font-bold block">+287% de velocidad</span>
        </div>

        {/* Jitter */}
        <div 
          onClick={() => setActiveMetric("jitter")}
          className={`border-l-2 pl-3 space-y-1 cursor-pointer transition-colors ${
            activeMetric === "jitter" ? "border-primary" : "border-neutral-800 hover:border-neutral-600"
          }`}
        >
          <span className="text-neutral-500 uppercase text-xs block font-medium">ESTABILIDAD (JITTER):</span>
          <div className="text-2xl sm:text-3xl font-black text-white tracking-tight">
            2 <span className="text-sm font-bold text-neutral-400">ms</span>
          </div>
          <span className="text-xs text-primary font-bold block">-91% de variación</span>
        </div>

        {/* Packet Loss */}
        <div className="border-l-2 border-neutral-800 pl-3 space-y-1">
          <span className="text-neutral-500 uppercase text-xs block font-medium">PÉRDIDA DE DATOS:</span>
          <div className="text-2xl sm:text-3xl font-black text-white tracking-tight">
            0.0 <span className="text-sm font-bold text-neutral-400">%</span>
          </div>
          <span className="text-xs text-emerald-400 font-bold block">0% pérdida • LIMPIO</span>
        </div>
      </div>

      {/* SVG Interactive Chart Box */}
      <div className="border-2 border-neutral-800 bg-black p-4 relative space-y-3">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-2 border-b border-neutral-800">
          <span className="text-xs font-mono text-neutral-300 font-bold uppercase">
            CURVA TEMPORAL: {current.label} ({current.unit})
          </span>

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

        <div className="w-full overflow-x-auto">
          <svg
            viewBox={`0 0 ${width} ${height}`}
            className="w-full h-44 sm:h-52 select-none"
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
                    fontSize="10"
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
                fontSize="10"
                fontFamily="monospace"
                textAnchor="middle"
              >
                {idx * 5}s
              </text>
            ))}

            {/* Before Path */}
            <path
              d={beforeSvgPath}
              fill="none"
              stroke="#EF4444"
              strokeWidth="2"
              strokeDasharray="4 3"
            />

            {/* After Path */}
            <path
              d={afterSvgPath}
              fill="none"
              stroke="#FF7F11"
              strokeWidth="3"
            />

            {/* Dots */}
            {beforeCoords.map((pt, idx) => (
              <circle
                key={`b-${idx}`}
                cx={pt.x}
                cy={pt.y}
                r="3"
                fill="#EF4444"
              />
            ))}

            {afterCoords.map((pt, idx) => (
              <circle
                key={`a-${idx}`}
                cx={pt.x}
                cy={pt.y}
                r={hoverIndex === idx ? 6 : 4}
                fill="#FF7F11"
                stroke="#000000"
                strokeWidth="1.5"
                onMouseEnter={() => setHoverIndex(idx)}
                onMouseLeave={() => setHoverIndex(null)}
                className="cursor-pointer transition-all"
              />
            ))}
          </svg>
        </div>

        {/* Minimal Explanation without bulky background */}
        <div className="border-l-2 border-primary pl-3 py-1 flex items-center justify-between text-xs font-sans">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-primary shrink-0" />
            <span className="text-neutral-300">
              <strong className="text-white">Diagnóstico: </strong>
              La línea naranja es continua y sin caídas porque se eliminó la congestión de canales en el aire.
            </span>
          </div>
          <span className="text-xs font-mono text-neutral-500 shrink-0">
            TEST #{testCount}
          </span>
        </div>
      </div>
    </div>
  );
}
