import "jsr:@supabase/functions-js/edge-runtime.d.ts"
import { createClient } from "jsr:@supabase/supabase-js@2"

type GenerateRequest = {
  nodeId?: string
  message?: string
  idempotencyKey?: string
  history?: AnthropicMessage[]
}

type AiRunStatus = "pending" | "succeeded" | "failed"

type AiMessageResponse = {
  id: string
  role: "assistant"
  content: string
  timestamp: string
}

type AnthropicMessage = {
  role: "user" | "assistant"
  content: string
}

type AnthropicResponse = {
  id: string
  type: string
  role: string
  content: Array<{ type: string; text: string }>
  model: string
  stop_reason: string
  usage: {
    input_tokens: number
    output_tokens: number
  }
}

type AiConfig = {
  base_url: string | null
  auth_token: string | null
}

const JSON_HEADERS = {
  "Content-Type": "application/json",
}

const CORS_HEADERS = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers": "authorization, x-client-info, apikey, content-type",
  "Access-Control-Allow-Methods": "POST, OPTIONS",
}

const SYSTEM_PROMPT = "你是卷了么 AI 助手。回答简洁、结构化，不编造信息。"

function jsonResponse(body: Record<string, unknown>, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      ...JSON_HEADERS,
      ...CORS_HEADERS,
    },
  })
}

function extractBearerToken(req: Request): string | null {
  const authHeader = req.headers.get("Authorization") ?? req.headers.get("authorization")
  if (!authHeader) return null
  const match = authHeader.match(/^Bearer\s+(.+)$/i)
  return match?.[1] ?? null
}

function isTerminalStatus(status: string): status is Extract<AiRunStatus, "succeeded" | "failed"> {
  return status === "succeeded" || status === "failed"
}

async function loadAiConfig(serviceClient: ReturnType<typeof createClient>): Promise<AiConfig> {
  const { data, error } = await serviceClient.schema("juanleme").rpc("get_ai_config")

  if (error) {
    throw new Error(`Failed to load AI config: ${error.message}`)
  }

  return data as AiConfig
}

async function callAnthropicApi(
  config: AiConfig,
  message: string,
  history: AnthropicMessage[]
): Promise<string> {
  if (!config.base_url || !config.auth_token) {
    return `收到你的消息：${message}\n\n已启用模板回复（尚未配置 AI 凭据）。请在 Vault 中配置 base_url 和 auth_token。`
  }

  const messages: AnthropicMessage[] = [
    ...history,
    { role: "user", content: message },
  ]

  const requestBody = {
    model: "claude-sonnet-4-20250514",
    max_tokens: 2048,
    system: SYSTEM_PROMPT,
    messages,
  }

  const response = await fetch(`${config.base_url}/v1/messages`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "x-api-key": config.auth_token,
      "anthropic-version": "2023-06-01",
    },
    body: JSON.stringify(requestBody),
  })

  if (!response.ok) {
    const errorText = await response.text()
    throw new Error(`Anthropic API error: ${response.status} ${errorText}`)
  }

  const data = (await response.json()) as AnthropicResponse
  const textContent = data.content.find((c) => c.type === "text")
  return textContent?.text ?? ""
}

Deno.serve(async (req: Request) => {
  if (req.method === "OPTIONS") {
    return new Response("ok", {
      status: 200,
      headers: CORS_HEADERS,
    })
  }

  if (req.method !== "POST") {
    return jsonResponse({ error: "Method not allowed" }, 405)
  }

  const supabaseUrl = Deno.env.get("SUPABASE_URL")
  const anonKey = Deno.env.get("SUPABASE_ANON_KEY")
  const serviceRoleKey = Deno.env.get("SUPABASE_SERVICE_ROLE_KEY")
  if (!supabaseUrl || !anonKey || !serviceRoleKey) {
    return jsonResponse({ error: "Supabase env is missing" }, 500)
  }

  const jwt = extractBearerToken(req)
  if (!jwt) {
    return jsonResponse({ error: "Missing bearer token" }, 401)
  }

  const jwtClient = createClient(supabaseUrl, anonKey, {
    global: {
      headers: {
        Authorization: `Bearer ${jwt}`,
      },
    },
  })

  const serviceClient = createClient(supabaseUrl, serviceRoleKey)

  let body: GenerateRequest
  try {
    body = (await req.json()) as GenerateRequest
  } catch {
    return jsonResponse({ error: "Invalid JSON body" }, 400)
  }

  const nodeId = body.nodeId?.trim()
  const message = body.message?.trim()
  const idempotencyKey = body.idempotencyKey?.trim() || undefined
  const history = body.history ?? []

  if (!nodeId || !message) {
    return jsonResponse({ error: "nodeId and message are required" }, 400)
  }

  const {
    data: { user },
    error: userError,
  } = await jwtClient.auth.getUser()

  if (userError || !user) {
    return jsonResponse({ error: userError?.message ?? "Unauthorized" }, 401)
  }

  const { data: nodeRows, error: nodeError } = await jwtClient
    .schema("juanleme").rpc("ai_get_workshop_node", { p_node_id: nodeId })

  if (nodeError || !nodeRows || nodeRows.length === 0) {
    return jsonResponse({ error: "Node not found" }, 404)
  }

  const workspaceId = nodeRows[0].workspace_id as string

  const { data: memberRows, error: memberError } = await jwtClient
    .schema("juanleme").rpc("ai_get_workspace_membership_me", { p_workspace_id: workspaceId })

  if (memberError) {
    return jsonResponse({ error: memberError.message }, 500)
  }

  if (!memberRows || memberRows.length === 0) {
    return jsonResponse({ error: "Forbidden" }, 403)
  }

  if (idempotencyKey) {
    const { data: existingRuns, error: idempotencyError } = await jwtClient
      .schema("juanleme").rpc("ai_get_run_by_idempotency_me", { p_idempotency_key: idempotencyKey })

    if (idempotencyError) {
      return jsonResponse({ error: idempotencyError.message }, 500)
    }

    if (existingRuns && existingRuns.length > 0) {
      const existingRun = existingRuns[0]
      if (isTerminalStatus(existingRun.status)) {
        const content = existingRun.status === "succeeded"
          ? existingRun.response ?? ""
          : `请求失败：${existingRun.error_message ?? "未知错误"}`
        const cached: AiMessageResponse = {
          id: existingRun.id,
          role: "assistant",
          content,
          timestamp: existingRun.updated_at ?? new Date().toISOString(),
        }
        return jsonResponse(cached, 200)
      }

      return jsonResponse({ error: "Duplicate idempotency key is still pending" }, 409)
    }
  }

  const { data: runId, error: insertError } = await serviceClient
    .schema("juanleme").rpc("ai_create_run_service", {
      p_workspace_id: workspaceId,
      p_node_id: nodeId,
      p_user_id: user.id,
      p_prompt: message,
      p_status: "pending",
      p_idempotency_key: idempotencyKey ?? null,
    })

  if (insertError || !runId) {
    if (idempotencyKey && insertError?.code === "23505") {
      return jsonResponse({ error: "Duplicate idempotency key" }, 409)
    }
    return jsonResponse({ error: insertError?.message ?? "Failed to create run" }, 500)
  }

  try {
    const aiConfig = await loadAiConfig(serviceClient)
    const aiContent = await callAnthropicApi(aiConfig, message, history)

    const { data: updated, error: updateError } = await serviceClient
      .schema("juanleme").rpc("ai_update_run_service", {
        p_run_id: runId,
        p_status: "succeeded",
        p_response: aiContent,
      })

    if (updateError || !updated) {
      return jsonResponse({ error: updateError?.message ?? "Failed to update run" }, 500)
    }

    const responseBody: AiMessageResponse = {
      id: runId,
      role: "assistant",
      content: aiContent,
      timestamp: new Date().toISOString(),
    }

    return jsonResponse(responseBody, 200)
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : "Unknown error"
    await serviceClient
      .schema("juanleme").rpc("ai_update_run_service", {
        p_run_id: runId,
        p_status: "failed",
        p_error_message: errorMessage,
      })

    return jsonResponse({ error: "Internal error", details: errorMessage }, 500)
  }
})
