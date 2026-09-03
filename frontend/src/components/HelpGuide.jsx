import React, { useState } from "react";
import {
  ChevronDown,
  ChevronUp
} from "lucide-react";

export function HelpGuide() {
  const [openQuestion, setOpenQuestion] = useState(null);

  const toggleQuestion = (index) => {
    setOpenQuestion(openQuestion === index ? null : index);
  };

  const simpleQuestions = [
    {
      icon: "📟",
      title: "¿Qué es el Router exactamente?",
      answer: "El router es el equipo principal de tu casa: recibe la señal de Internet desde el cable exterior y la distribuye a tus celulares, computadoras, televisores y demás dispositivos. También administra la red local, asigna una dirección IP a cada equipo y controla qué tráfico entra o sale. Por eso es el punto central para consultar la conexión y revisar la seguridad de tu hogar."
    },
    {
      icon: "🔢",
      title: "¿Qué es la Dirección IP (ej. 192.168.1.1)?",
      answer: "La dirección IP funciona como el número de puerta o teléfono de cada equipo en tu hogar. Permite que el router identifique qué teléfono o computadora solicitó cada página o video sin confusiones. La dirección 192.168.1.1 suele ser la dirección privada del propio router: solo se utiliza dentro de tu red y no identifica públicamente tu casa en Internet. Cada dispositivo conectado recibe otra dirección privada diferente."
    },
    {
      icon: "💾",
      title: "¿Qué es el Firmware?",
      answer: "El firmware es el software interno del router que gestiona su funcionamiento básico, la red Wi-Fi, las conexiones y parte de la seguridad. Es parecido al sistema operativo del equipo. Una versión antigua puede seguir funcionando, pero conviene revisar si el fabricante publicó actualizaciones y evitar exponer la administración del router a Internet. No instales firmware sin confirmar que corresponde exactamente a tu modelo y revisión de hardware."
    },
    {
      icon: "📶",
      title: "¿Cuál es la diferencia entre Wi-Fi y Cable Ethernet?",
      answer: "El Wi-Fi transmite datos por el aire y permite moverse por la casa, aunque la distancia, las paredes y otras redes pueden reducir su velocidad o estabilidad. El cable Ethernet conecta directamente el equipo al router y normalmente ofrece menor latencia, menos interferencias y una conexión más constante. Para videollamadas, juegos o transferencias grandes, el cable suele ser la opción más estable."
    },
    {
      icon: "💡",
      title: "¿Qué hacer si la conexión se interrumpe?",
      answer: "Primero comprueba si el problema afecta a todos los dispositivos o solo a uno. Revisa que los cables y las luces del router estén conectados correctamente; después, desconéctalo de la corriente, espera 15 segundos y vuelve a conectarlo. El servicio suele restablecerse en aproximadamente dos minutos. Si el problema continúa, consulta el estado WAN, prueba con otro cable y contacta a tu proveedor. No pulses el botón de restablecimiento de fábrica: borra la configuración."
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
