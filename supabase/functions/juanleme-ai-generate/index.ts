import "jsr:@supabase/functions-js/edge-runtime.d.ts"
import { createClient } from "jsr:@supabase/supabase-js@2"

type GenerateRequest = {
  nodeId?: string
  message?: string
  idempotencyKey?: string
}

type AiRunStatus = "pending" | "succeeded" | "failed"

type AiMessageResponse = {
  id: string
  role: "assistant"
  content: string
  timestamp: string
}

const JSON_HEADERS = {
  "Content-Type": "application/json",
}

const CORS_HEADERS = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers": "authorization, x-client-info, apikey, content-type",
  "Access-Control-Allow-Methods": "POST, OPTIONS",
}

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

function buildTemplateReply(message: string, hasApiKey: boolean): string {
  if (!hasApiKey) {
    return `收到你的消息：${message}\n\n已启用模板回复（尚未配置 AI_API_KEY）。请稍后在环境变量中接入真实模型。`
  }
  return `我理解你的输入是：“${message}”。当前仍使用占位模板，后续会切换到真实 AI Provider。`
}

function isTerminalStatus(status: string): status is Extract<AiRunStatus, "succeeded" | "failed"> {
  return status === "succeeded" || status === "failed"
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

  const { data: lineage, error: lineageError } = await jwtClient
    .schema("juanleme")
    .from("workshop_nodes")
    .select("id, projects!inner(workspaces!inner(id))")
    .eq("id", nodeId)
    .single()

  if (lineageError || !lineage) {
    return jsonResponse({ error: "Node not found" }, 404)
  }

  const workspaceId = (lineage as { projects: { workspaces: { id: string } } }).projects.workspaces.id

  const { data: membership, error: memberError } = await jwtClient
    .schema("juanleme")
    .from("workspace_memberships")
    .select("user_id")
    .eq("workspace_id", workspaceId)
    .eq("user_id", user.id)
    .maybeSingle()

  if (memberError) {
    return jsonResponse({ error: memberError.message }, 500)
  }

  if (!membership) {
    return jsonResponse({ error: "Forbidden" }, 403)
  }

  if (idempotencyKey) {
    const { data: existingRun, error: idempotencyError } = await jwtClient
      .schema("juanleme")
      .from("ai_runs")
      .select("id, status, response, error_message, updated_at")
      .eq("user_id", user.id)
      .eq("idempotency_key", idempotencyKey)
      .maybeSingle()

    if (idempotencyError) {
      return jsonResponse({ error: idempotencyError.message }, 500)
    }

    if (existingRun) {
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

  const { data: insertedRun, error: insertError } = await serviceClient
    .schema("juanleme")
    .from("ai_runs")
    .insert({
      workspace_id: workspaceId,
      node_id: nodeId,
      user_id: user.id,
      prompt: message,
      status: "pending",
      idempotency_key: idempotencyKey ?? null,
    })
    .select("id")
    .single()

  if (insertError || !insertedRun) {
    if (idempotencyKey && insertError?.code === "23505") {
      return jsonResponse({ error: "Duplicate idempotency key" }, 409)
    }
    return jsonResponse({ error: insertError?.message ?? "Failed to create run" }, 500)
  }

  const runId = insertedRun.id as string

  try {
    const hasApiKey = Boolean(Deno.env.get("AI_API_KEY"))
    const aiContent = buildTemplateReply(message, hasApiKey)

    const { error: updateError } = await serviceClient
      .schema("juanleme")
      .from("ai_runs")
      .update({
        status: "succeeded",
        response: aiContent,
        updated_at: new Date().toISOString(),
      })
      .eq("id", runId)

    if (updateError) {
      return jsonResponse({ error: updateError.message }, 500)
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
      .schema("juanleme")
      .from("ai_runs")
      .update({
        status: "failed",
        error_message: errorMessage,
        updated_at: new Date().toISOString(),
      })
      .eq("id", runId)

    return jsonResponse({ error: "Internal error", details: errorMessage }, 500)
  }
})
