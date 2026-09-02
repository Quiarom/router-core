// Command router-core-agent implementa la capa de razonamiento de router-core.
// Consulta exclusivamente la API local de observación y usa MiniMax M3 mediante
// OpenRouter para responder preguntas sobre el router.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	version         = "0.2.0"
	maxQuestionSize = 64 << 10
)

type options struct {
	routerCoreURL           string
	openrouterURL           string
	openrouterModel         string
	openrouterFallbackModel string
	openrouterKeyEnv        string
	question                string
	serveAddr               string
	dryRun                  bool
	timeout                 time.Duration
}

type chatInput struct {
	Question string `json:"question"`
}

type observationStep struct {
	Tool       string          `json:"tool"`
	Path       string          `json:"path"`
	HTTPStatus int             `json:"http_status"`
	Result     json.RawMessage `json:"result"`
}

type agentResult struct {
	Answer string            `json:"answer"`
	Model  string            `json:"model"`
	Mode   string            `json:"mode"`
	Steps  []observationStep `json:"steps"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func main() {
	opts, err := parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "router-core-agent:", err)
		os.Exit(2)
	}
	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "router-core-agent:", err)
		os.Exit(1)
	}
}

func parseFlags(args []string) (options, error) {
	fs := flag.NewFlagSet("router-core-agent", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	opts := options{
		routerCoreURL: "http://127.0.0.1:8484",
		// Default: GMI Cloud direct (api.gmi-serving.com). Override
		// with --openrouter-url to use OpenRouter or any other
		// OpenAI-compatible chat-completions endpoint.
		openrouterURL: "https://api.gmi-serving.com/v1/chat/completions",
		// Default model: MiniMax M3 served by GMICloud. Override
		// with --model to use a different one (e.g. M2.7 for lower
		// latency, or any other chat-completions-compatible model).
		openrouterModel: envOrDefault("GMI_MODEL", "MiniMaxAI/MiniMax-M3"),
		// Fallback model: M2.7. If M3 returns a transient error
		// (5xx, timeout, connection reset), the agent retries
		// once with this model. Override with GMI_FALLBACK_MODEL
		// to a different chat-completions model.
		openrouterFallbackModel: envOrDefault("GMI_FALLBACK_MODEL", "MiniMaxAI/MiniMax-M2.7"),
		// Default key env var: GMI_SERVING_API_KEY. Falls back to
		// OPENROUTER_API_KEY for backward compatibility with the
		// OpenRouter path.
		openrouterKeyEnv: envOrDefault("GMI_KEY_ENV", "GMI_SERVING_API_KEY"),
		timeout:          45 * time.Second,
	}
	fs.StringVar(&opts.routerCoreURL, "router-core-url", opts.routerCoreURL, "URL local de router-core serve")
	fs.StringVar(&opts.openrouterURL, "openrouter-url", opts.openrouterURL, "URL de Chat Completions (cualquier endpoint compatible con OpenAI)")
	fs.StringVar(&opts.openrouterModel, "model", opts.openrouterModel, "identificador del modelo (e.g. MiniMaxAI/MiniMax-M3 o minimax/minimax-m3:free)")
	fs.StringVar(&opts.openrouterFallbackModel, "model-fallback", opts.openrouterFallbackModel, "modelo de fallback si M3 falla (e.g. MiniMaxAI/MiniMax-M2.7)")
	fs.StringVar(&opts.openrouterKeyEnv, "key-env", opts.openrouterKeyEnv, "variable de entorno que contiene la clave")
	fs.StringVar(&opts.question, "question", "", "pregunta del usuario, o - para leer stdin")
	fs.StringVar(&opts.serveAddr, "serve", "", "expone la API del chat en esta dirección loopback, por ejemplo 127.0.0.1:8585")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "usa el agente determinista sin llamar a OpenRouter")
	fs.DurationVar(&opts.timeout, "timeout", opts.timeout, "tiempo máximo por consulta")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	return opts, nil
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func run(opts options) error {
	if err := validateLoopbackURL(opts.routerCoreURL); err != nil {
		return fmt.Errorf("URL de router-core inválida: %w", err)
	}
	if opts.serveAddr != "" {
		return runAgentServer(opts)
	}

	question := opts.question
	if question == "-" {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, maxQuestionSize))
		if err != nil {
			return fmt.Errorf("leer pregunta desde stdin: %w", err)
		}
		question = strings.TrimSpace(string(data))
	}
	if question == "" {
		return errors.New("la pregunta es obligatoria; usa --question o stdin")
	}

	result, err := executeQuestion(context.Background(), opts, question)
	if err != nil {
		return err
	}
	fmt.Println(result.Answer)
	return nil
}

func runAgentServer(opts options) error {
	if !isLoopbackAddr(opts.serveAddr) {
		return fmt.Errorf("se rechaza --serve %q: solo se permite loopback", opts.serveAddr)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", withLocalCORS(healthzHandler(opts)))
	mux.HandleFunc("/v0/chat", withLocalCORS(chatHandler(opts)))

	server := &http.Server{
		Addr:              opts.serveAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      opts.timeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
	}
	logf("API de chat disponible en http://%s/v0/chat", opts.serveAddr)
	logf("router-core=%s model=%s", opts.routerCoreURL, opts.openrouterModel)
	if opts.dryRun || os.Getenv(opts.openrouterKeyEnv) == "" {
		logf("modo determinista; configura %s para usar MiniMax M3", opts.openrouterKeyEnv)
	}
	return server.ListenAndServe()
}

// healthzHandler answers GET /healthz with the live model
// name and a state of "ok". Exported (lowercase) so the
// test file can call it directly.
func healthzHandler(opts options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "método no permitido"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"state": "ok",
			"model": opts.openrouterModel,
		})
	}
}

func chatHandler(opts options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "método no permitido"})
			return
		}

		reader := http.MaxBytesReader(w, r.Body, maxQuestionSize)
		defer reader.Close()
		var input chatInput
		if err := json.NewDecoder(reader).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "cuerpo JSON inválido"})
			return
		}
		input.Question = strings.TrimSpace(input.Question)
		if input.Question == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "la pregunta no puede estar vacía"})
			return
		}

		result, err := executeQuestion(r.Context(), opts, input.Question)
		if err != nil {
			logf("consulta fallida: %v", err)
			writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func withLocalCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !isLocalOrigin(origin) {
				writeJSON(w, http.StatusForbidden, errorResponse{Error: "origen no permitido"})
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		next(w, r)
	}
}

func isLocalOrigin(rawOrigin string) bool {
	parsed, err := url.Parse(rawOrigin)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validateLoopbackURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" {
		return errors.New("el esquema debe ser http")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("la URL no puede incluir credenciales, query ni fragmento")
	}
	host := parsed.Hostname()
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("el host debe ser loopback")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func executeQuestion(parent context.Context, opts options, question string) (agentResult, error) {
	ctx, cancel := context.WithTimeout(parent, opts.timeout)
	defer cancel()

	routerClient := newRouterCoreClient(opts.routerCoreURL, opts.timeout)
	device, err := routerClient.get(ctx, "/v0/device")
	if err != nil {
		return agentResult{}, fmt.Errorf("consultar dispositivo: %w", err)
	}
	status, statusErr := routerClient.get(ctx, "/v0/status")
	if statusErr != nil {
		status = observation{Path: "/v0/status", Status: http.StatusServiceUnavailable, Body: json.RawMessage(`{"state":"unavailable"}`)}
	}
	capabilities, err := routerClient.get(ctx, "/v0/capabilities")
	if err != nil {
		return agentResult{}, fmt.Errorf("consultar capacidades: %w", err)
	}
	clients, clientsErr := routerClient.get(ctx, "/v0/clients")
	if clientsErr != nil {
		clients = observation{
			Path:   "/v0/clients",
			Status: http.StatusServiceUnavailable,
			Body:   json.RawMessage(`{"state":"unavailable"}`),
		}
	}

	steps := []observationStep{
		stepFromObservation("get_device", device),
		stepFromObservation("get_status", status),
		stepFromObservation("get_clients", clients),
		stepFromObservation("get_capabilities", capabilities),
	}
	logf("pregunta=%q", question)

	apiKey := strings.TrimSpace(os.Getenv(opts.openrouterKeyEnv))
	if opts.dryRun || apiKey == "" {
		return runStub(ctx, routerClient, opts.openrouterModel, question, steps)
	}
	return runLive(ctx, opts, routerClient, apiKey, device.Body, status.Body, clients.Body, capabilities.Body, question, steps)
}

func stepFromObservation(toolName string, value observation) observationStep {
	return observationStep{
		Tool:       toolName,
		Path:       value.Path,
		HTTPStatus: value.Status,
		Result:     value.Body,
	}
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "router-core-agent: "+format+"\n", args...)
}

type observation struct {
	Path   string
	Status int
	Body   json.RawMessage
}

type routerCoreClient struct {
	baseURL string
	http    *http.Client
}

func newRouterCoreClient(baseURL string, timeout time.Duration) *routerCoreClient {
	return &routerCoreClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *routerCoreClient) get(ctx context.Context, path string) (observation, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return observation{}, err
	}
	response, err := c.http.Do(req)
	if err != nil {
		return observation{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return observation{}, err
	}
	if len(body) == 0 {
		body = []byte(fmt.Sprintf(`{"http_status":%d}`, response.StatusCode))
	}
	if !json.Valid(body) {
		return observation{}, fmt.Errorf("%s devolvió una respuesta que no es JSON", path)
	}
	return observation{Path: path, Status: response.StatusCode, Body: json.RawMessage(body)}, nil
}

func compactJSON(body json.RawMessage) string {
	if len(body) == 0 {
		return "<absent>"
	}
	var output bytes.Buffer
	if err := json.Compact(&output, body); err != nil {
		return string(body)
	}
	return output.String()
}

type functionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type openRouterTool struct {
	Type     string             `json:"type"`
	Function functionDefinition `json:"function"`
}

var clientsTool = openRouterTool{
	Type: "function",
	Function: functionDefinition{
		Name:        "get_clients",
		Description: "Consulta GET /v0/clients en router-core y devuelve las concesiones DHCP observadas. No determina por sí sola que un equipo sea confiable.",
		Parameters: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		},
	},
}

var securityTool = openRouterTool{
	Type: "function",
	Function: functionDefinition{
		Name:        "get_security",
		Description: "Consulta GET /v0/security/<name> en router-core. Devuelve una observación estructurada y nunca modifica el router.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type": "string",
					"enum": []string{"wireless", "wps", "dmz", "upnp", "remote-management", "forwarding"},
				},
			},
			"required":             []string{"name"},
			"additionalProperties": false,
		},
	},
}

func buildSystemPrompt(device, status, clients, capabilities json.RawMessage) string {
	return fmt.Sprintf(`Eres un auditor de redes domésticas de solo lectura.
Responde siempre en español claro y basa cada afirmación exclusivamente en las observaciones entregadas.
Nunca inventes valores ni asegures que un equipo es legítimo solo por tener una concesión DHCP.
Distingue verified, absent, unsupported_or_unverified y unavailable. No los conviertas en verdaderos o falsos.
Puedes pedir exactamente una observación por turno con get_clients o get_security. Cuando tengas evidencia suficiente, responde sin más herramientas.
Resume: hechos observados, límites de la evidencia, posibles riesgos y recomendaciones. Nunca afirmes que realizaste cambios.

DISPOSITIVO
%s

ESTADO
%s

APARATOS OBSERVADOS
%s

CAPACIDADES
%s`, compactJSON(device), compactJSON(status), compactJSON(clients), compactJSON(capabilities))
}

type toolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openRouterToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function toolCallFunction `json:"function"`
}

type message struct {
	Role       string               `json:"role"`
	Content    string               `json:"content,omitempty"`
	ToolCalls  []openRouterToolCall `json:"tool_calls,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
}

type chatRequest struct {
	Model    string           `json:"model"`
	Messages []message        `json:"messages"`
	Tools    []openRouterTool `json:"tools"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// runLive calls the chat-completions endpoint, looping on tool
// calls until the model emits a final answer. If the first model
// returns a transient error (5xx, timeout, connection reset),
// the function retries once with the fallback model. The final
// answer reports which model actually answered.
func runLive(ctx context.Context, opts options, routerClient *routerCoreClient, apiKey string, device, status, clients, capabilities json.RawMessage, question string, steps []observationStep) (agentResult, error) {
	result, err := runLiveOnce(ctx, opts, routerClient, apiKey, device, status, clients, capabilities, question, steps, opts.openrouterModel)
	if err == nil {
		return result, nil
	}
	// Retry once with the fallback model if the primary error
	// looks transient. Skip the retry for structural errors
	// (bad URL, missing key, parse failure).
	if !isTransient(err) {
		return agentResult{}, err
	}
	if opts.openrouterFallbackModel == "" || opts.openrouterFallbackModel == opts.openrouterModel {
		return agentResult{}, err
	}
	logf("modelo primario falló (%v): reintentando con %s", opts.openrouterModel, opts.openrouterFallbackModel)
	result, err = runLiveOnce(ctx, opts, routerClient, apiKey, device, status, clients, capabilities, question, steps, opts.openrouterFallbackModel)
	if err != nil {
		return agentResult{}, fmt.Errorf("modelo primario y fallback fallaron: %w", err)
	}
	return result, nil
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "HTTP 5") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "connection refused") ||
		strings.Contains(s, "EOF")
}

func runLiveOnce(ctx context.Context, opts options, routerClient *routerCoreClient, apiKey string, device, status, clients, capabilities json.RawMessage, question string, steps []observationStep, model string) (agentResult, error) {
	history := []message{
		{Role: "system", Content: buildSystemPrompt(device, status, clients, capabilities)},
		{Role: "user", Content: question},
	}
	modelClient := &http.Client{Timeout: opts.timeout}

	for range 8 {
		requestBody, err := json.Marshal(chatRequest{
			Model:    model,
			Messages: history,
			Tools:    []openRouterTool{clientsTool, securityTool},
		})
		if err != nil {
			return agentResult{}, fmt.Errorf("codificar solicitud al modelo: %w", err)
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, opts.openrouterURL, bytes.NewReader(requestBody))
		if err != nil {
			return agentResult{}, err
		}
		request.Header.Set("Authorization", "Bearer "+apiKey)
		request.Header.Set("Content-Type", "application/json")
		// OpenRouter-specific tracking headers. These are no-ops
		// on api.gmi-serving.com; including them there is harmless
		// but pointless. Kept for OpenRouter compatibility.
		if strings.Contains(opts.openrouterURL, "openrouter.ai") {
			request.Header.Set("HTTP-Referer", "https://github.com/Quiarom/router-core")
			request.Header.Set("X-Title", "router-core")
		}

		response, err := modelClient.Do(request)
		if err != nil {
			return agentResult{}, fmt.Errorf("conectar con OpenRouter: %w", err)
		}
		responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
		response.Body.Close()
		if readErr != nil {
			return agentResult{}, fmt.Errorf("leer respuesta de OpenRouter: %w", readErr)
		}
		if response.StatusCode/100 != 2 {
			return agentResult{}, fmt.Errorf("el endpoint respondió HTTP %d: %s", response.StatusCode, compactJSON(responseBody))
		}

		var parsed chatResponse
		if err := json.Unmarshal(responseBody, &parsed); err != nil {
			return agentResult{}, fmt.Errorf("decodificar respuesta de OpenRouter: %w", err)
		}
		if parsed.Error != nil {
			return agentResult{}, fmt.Errorf("OpenRouter: %s", parsed.Error.Message)
		}
		if len(parsed.Choices) == 0 {
			return agentResult{}, errors.New("OpenRouter no devolvió alternativas")
		}

		assistantMessage := parsed.Choices[0].Message
		history = append(history, assistantMessage)
		if len(assistantMessage.ToolCalls) == 0 {
			answer := strings.TrimSpace(assistantMessage.Content)
			if answer == "" {
				return agentResult{}, errors.New("MiniMax devolvió una respuesta vacía")
			}
			return agentResult{Answer: answer, Model: model, Mode: "live", Steps: steps}, nil
		}

		for _, call := range assistantMessage.ToolCalls {
			path := ""
			if call.Function.Name == "get_clients" {
				path = "/v0/clients"
			} else if call.Function.Name == "get_security" {
				var arguments struct {
					Name string `json:"name"`
				}
				if err := json.Unmarshal([]byte(call.Function.Arguments), &arguments); err != nil {
					return agentResult{}, fmt.Errorf("argumentos inválidos para get_security: %w", err)
				}
				if !isAllowedSecurityCapability(arguments.Name) {
					return agentResult{}, fmt.Errorf("capacidad de seguridad no permitida: %q", arguments.Name)
				}
				path = "/v0/security/" + arguments.Name
			} else {
				return agentResult{}, fmt.Errorf("herramienta no permitida: %s", call.Function.Name)
			}
			observed, err := routerClient.get(ctx, path)
			if err != nil {
				observed = observation{Path: path, Status: http.StatusServiceUnavailable, Body: mustJSON(errorResponse{Error: err.Error()})}
			}
			steps = append(steps, stepFromObservation(call.Function.Name, observed))
			logf("%s -> HTTP %d", call.Function.Name, observed.Status)
			history = append(history, message{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    string(observed.Body),
			})
		}
	}
	return agentResult{}, errors.New("el agente superó ocho turnos sin producir una respuesta final")
}

func isAllowedSecurityCapability(name string) bool {
	switch name {
	case "wireless", "wps", "dmz", "upnp", "remote-management", "forwarding":
		return true
	default:
		return false
	}
}

func mustJSON(value any) json.RawMessage {
	body, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{"error":"no se pudo codificar el error"}`)
	}
	return json.RawMessage(body)
}

func stubSequence(question string) []string {
	query := strings.ToLower(question)
	switch {
	case strings.Contains(query, "wi-fi"), strings.Contains(query, "wifi"), strings.Contains(query, "inalámbrica"), strings.Contains(query, "expuest"):
		return []string{"wireless", "wps", "remote-management"}
	case strings.Contains(query, "quién"), strings.Contains(query, "quien"), strings.Contains(query, "conectad"), strings.Contains(query, "dispositiv"), strings.Contains(query, "aparato"):
		return []string{"wireless", "wps", "remote-management", "dmz", "upnp", "forwarding"}
	default:
		return []string{"dmz", "forwarding", "upnp"}
	}
}

func runStub(ctx context.Context, routerClient *routerCoreClient, model, question string, steps []observationStep) (agentResult, error) {
	sequence := stubSequence(question)
	for _, name := range sequence {
		observed, err := routerClient.get(ctx, "/v0/security/"+name)
		if err != nil {
			return agentResult{}, fmt.Errorf("consultar seguridad %s: %w", name, err)
		}
		steps = append(steps, stepFromObservation("get_security", observed))
	}

	answer := "Modo de demostración: revisé las observaciones disponibles en router-core. " +
		"La API de MiniMax no está configurada, así que no debo elaborar una conclusión inteligente ni inventar datos. " +
		"Configura OPENROUTER_API_KEY y vuelve a consultar para obtener el análisis de MiniMax M3."
	return agentResult{Answer: answer, Model: model, Mode: "stub", Steps: steps}, nil
}
