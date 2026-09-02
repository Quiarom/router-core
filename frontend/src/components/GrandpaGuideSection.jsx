import React, { useState } from "react";
import { 
  HelpCircle, 
  ChevronDown, 
  ChevronUp, 
  Power
} from "lucide-react";

export function GrandpaGuideSection({ clientCount = 3, isWanConnected = true }) {
  const [openQuestion, setOpenQuestion] = useState(null);

  const toggleQuestion = (index) => {
    setOpenQuestion(openQuestion === index ? null : index);
  };

  const simpleQuestions = [
    {
      icon: "📟",
      title: "¿Qué es el Router exactamente?",
      answer: "El Router es la cajita con luces y antenas que te instalaron en la casa. Imagínate que es como el cartero del barrio: recibe las cartas (los datos) que vienen por el cable desde la calle y te las reparte por el aire a tu celular y a la televisión."
    },
    {
      icon: "🔢",
      title: "¿Qué es la Dirección IP (ese número 192.168.1.1)?",
      answer: "La dirección IP es como el número de teléfono o el número de la puerta de tu casa, pero para las computadoras. Cada aparato (tu teléfono, tu tele, el router) tiene su propio numerito para que no se confundan cuando te mandan un video o una foto de WhatsApp."
    },
    {
      icon: "💾",
      title: "¿Qué es el Firmware?",
      answer: "El firmware es el cerebro que está grabado adentro del aparato. Es como el motor de un auto o el sistema que hace prender las luces. Mientras esté en orden y sin errores, no tienes que preocuparte de nada."
    },
    {
      icon: "📶",
      title: "¿Cuál es la diferencia entre Wi-Fi y Cable?",
      answer: "El Wi-Fi es el internet que viaja por el aire de forma invisible (como la música en la radio). Si te alejas mucho o hay muchas paredes gruesas, la señal llega más débil. El cable es un cable de plástico que va directo del router a la tele o a la compu para que nunca se corte."
    },
    {
      icon: "💡",
      title: "¿Qué hago si algún día no me anda el internet?",
      answer: "¡No te preocupes! El truco que siempre funciona es desenchufar el router del tomacorriente de la pared, contar despacito hasta 15, y volver a enchufarlo. Tarda unos dos minutos en prender todas las lucecitas y casi siempre se arregla solo."
    }
  ];

  return (
    <div className="w-full border-2 border-neutral-800 bg-neutral-950 p-4 sm:p-5 shadow-sm space-y-5 text-white">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b-2 border-neutral-800 pb-5 pt-1">
        <div>
          <h2 className="text-xl sm:text-2xl font-black uppercase tracking-tight text-white font-sans">
            Tu internet claramente
          </h2>
        </div>
      </div>

      {/* Minimal Telemetry Columns for 80-year-old clarity */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-6 pt-1 text-xs">
        {/* Item 1 */}
        <div className="border-l-2 border-emerald-500 pl-3 space-y-1">
          <span className="text-neutral-500 uppercase text-xs block font-medium font-mono">¿HAY INTERNET HOY?:</span>
          <span className="font-bold text-emerald-400 text-sm block font-sans">
            {isWanConnected ? "SÍ, PERFECTO" : "SIN SEÑAL"}
          </span>
        </div>

        {/* Item 2 */}
        <div className="border-l-2 border-emerald-500 pl-3 space-y-1">
          <span className="text-neutral-500 uppercase text-xs block font-medium font-mono">¿PELIGROS O INTRUSOS?:</span>
          <span className="font-bold text-white text-sm block font-sans">TODO SEGURO</span>
        </div>

        {/* Item 3 */}
        <div className="border-l-2 border-neutral-800 pl-3 space-y-1">
          <span className="text-neutral-500 uppercase text-xs block font-medium font-mono">APARATOS EN CASA:</span>
          <span className="font-bold text-white text-sm block font-sans">{clientCount} CONECTADOS</span>
        </div>

        {/* Item 4 */}
        <div className="border-l-2 border-neutral-800 pl-3 space-y-1">
          <span className="text-neutral-500 uppercase text-xs block font-medium font-mono">VELOCIDAD:</span>
          <span className="font-bold text-primary text-sm block font-sans">95 MEGAS (MUY RÁPIDO)</span>
        </div>
      </div>

      {/* Dictionary */}
      <div className="border-t border-neutral-800 pt-4 space-y-2">
        <div className="flex items-center gap-2 pb-2">
          <HelpCircle className="h-4 w-4 text-primary" />
          <h4 className="text-xs font-mono uppercase font-bold text-white">
            DICCIONARIO SENCILLO: ¿QUÉ SIGNIFICA CADA COSA?
          </h4>
        </div>

        <div className="divide-y divide-neutral-800">
          {simpleQuestions.map((q, idx) => (
            <div key={idx} className="py-2.5">
              <button
                onClick={() => toggleQuestion(idx)}
                className="w-full flex items-center justify-between text-left py-1 hover:text-primary transition-colors cursor-pointer group"
              >
                <span className="text-xs sm:text-sm font-bold flex items-center gap-2.5 text-neutral-200 group-hover:text-primary">
                  <span>{q.icon}</span>
                  <span>{q.title}</span>
                </span>
                <span className="text-xs text-neutral-500 font-mono">
                  {openQuestion === idx ? (
                    <ChevronUp className="h-4 w-4 text-primary" />
                  ) : (
                    <ChevronDown className="h-4 w-4 text-neutral-500" />
                  )}
                </span>
              </button>

              {openQuestion === idx && (
                <div className="mt-2 text-sm text-neutral-300 pl-4 pr-2 leading-relaxed border-l-2 border-primary font-sans">
                  {q.answer}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Golden Rule minimal */}
      <div className="border-l-2 border-primary pl-3 py-2 flex items-center gap-3 text-xs">
        <Power className="h-4 w-4 text-primary shrink-0" />
        <div className="text-neutral-300 font-sans">
          <strong className="text-white font-mono uppercase">El consejo de oro: </strong>
          Si algún día no te abre una página, desenchufa el router de la pared por 15 segundos y vuélvelo a enchufar.
        </div>
      </div>
    </div>
  );
}
