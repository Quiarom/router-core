import React, { useState } from "react";
import { 
  ChevronDown, 
  ChevronUp
} from "lucide-react";

export function HelpGuide({ clientCount = 3, isWanConnected = true }) {
  const [openQuestion, setOpenQuestion] = useState(null);

  const toggleQuestion = (index) => {
    setOpenQuestion(openQuestion === index ? null : index);
  };

  const simpleQuestions = [
    {
      icon: "📟",
      title: "¿Qué es el Router exactamente?",
      answer: "El router es el equipo principal de tu casa: recibe la señal de Internet desde el cable exterior y la distribuye inalámbricamente por Wi-Fi a tus celulares, computadoras y televisores."
    },
    {
      icon: "🔢",
      title: "¿Qué es la Dirección IP (ej. 192.168.1.1)?",
      answer: "La dirección IP funciona como el número de puerta o teléfono de cada equipo en tu hogar. Permite que el router identifique qué teléfono o computadora solicitó cada página o video sin confusiones."
    },
    {
      icon: "💾",
      title: "¿Qué es el Firmware?",
      answer: "El firmware es el software interno del router que gestiona su funcionamiento básico y seguridad. Mientras opere de forma estable y sin errores, no requiere intervención manual."
    },
    {
      icon: "📶",
      title: "¿Cuál es la diferencia entre Wi-Fi y Cable Ethernet?",
      answer: "El Wi-Fi transmite datos por el aire para mayor comodidad y movilidad. El cable Ethernet conecta directamente el equipo al router, proporcionando la máxima estabilidad y menor latencia posible."
    },
    {
      icon: "💡",
      title: "¿Qué hacer si la conexión se interrumpe?",
      answer: "La solución más efectiva es reiniciar el router: desconéctalo de la corriente eléctrica, espera 15 segundos y vuelve a conectarlo. En aproximadamente dos minutos el servicio se restablecerá automáticamente."
    }
  ];

  return (
    <div className="w-full border-2 border-neutral-800 bg-neutral-950 p-4 sm:p-5 shadow-sm space-y-4 text-white font-sans">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b-2 border-neutral-800 pb-4 pt-1">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="text-xl sm:text-2xl font-black uppercase tracking-tight text-white font-sans">
              Guía Rápida de Red
            </h2>
          </div>
          <p className="text-xs text-neutral-400 font-mono mt-0.5">
            Conceptos clave explicados sin rodeos para entender el estado de tu conexión.
          </p>
        </div>

        {/* Quick summary status */}
        <div className="flex items-center gap-2 text-xs font-mono">
          <span className={`h-2 w-2 rounded-full ${isWanConnected ? "bg-emerald-500 animate-pulse" : "bg-rose-500"}`} />
          <span className="text-neutral-300 font-bold uppercase">
            {isWanConnected ? "SERVICIO ESTABLE" : "SIN CONEXIÓN"}
          </span>
          <span className="text-neutral-600">|</span>
          <span className="text-neutral-400 font-bold">{clientCount} EQUIPOS</span>
        </div>
      </div>

      {/* Accordion Questions */}
      <div className="space-y-2.5">
        {simpleQuestions.map((q, idx) => {
          const isOpen = openQuestion === idx;

          return (
            <div 
              key={idx}
              className={`border-2 transition-colors ${
                isOpen 
                  ? "border-primary bg-black" 
                  : "border-neutral-800 bg-neutral-900/60 hover:border-neutral-700"
              }`}
            >
              <button
                type="button"
                onClick={() => toggleQuestion(idx)}
                className="w-full flex items-center justify-between text-left p-3.5 cursor-pointer"
              >
                <div className="flex items-center gap-3 pr-2">
                  <span className="text-base select-none">{q.icon}</span>
                  <span className="text-xs sm:text-sm font-bold uppercase tracking-wide text-white font-mono">
                    {q.title}
                  </span>
                </div>
                <div className="text-neutral-400 shrink-0">
                  {isOpen ? (
                    <ChevronUp className="h-4 w-4 text-primary" />
                  ) : (
                    <ChevronDown className="h-4 w-4" />
                  )}
                </div>
              </button>

              {isOpen && (
                <div className="px-4 pb-4 pt-2 text-xs sm:text-sm text-neutral-300 leading-relaxed border-t border-neutral-800 font-sans">
                  {q.answer}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default HelpGuide;
