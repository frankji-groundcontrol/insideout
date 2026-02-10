import "jsr:@supabase/functions-js/edge-runtime.d.ts"
import { createClient } from "jsr:@supabase/supabase-js@2"

type ExportFormat = "markdown" | "html"
type JobStatus = "pending" | "succeeded" | "failed"

type ExportRequest = {
  workshopId?: string
  format?: ExportFormat
  includeEmptyNodes?: boolean
  idempotencyKey?: string
}

type WorkspaceRow = {
  project_id: string
  workspace_id: string
  title: string
  description: string
}

type NodeRow = {
  id: string
  title: string
  description: string
  order: number
  content: unknown
}

type ExportJobRow = {
  id: string
  status: JobStatus
  format: string
  created_at: string
  artifact_url?: string | null
  artifact_path?: string | null
  error_message?: string | null
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

function normalizeContent(value: unknown): string {
  if (typeof value === "string") {
    return value.trim()
  }
  if (value === null || value === undefined) {
    return ""
  }
  if (typeof value === "object") {
    const record = value as Record<string, unknown>
    if (typeof record.text === "string") {
      return record.text.trim()
    }
    try {
      return JSON.stringify(value, null, 2)
    } catch {
      return ""
    }
  }
  return String(value)
}

function isTerminalStatus(status: string): status is Extract<JobStatus, "succeeded" | "failed"> {
  return status === "succeeded" || status === "failed"
}

function escapeHtml(raw: string): string {
  return raw
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;")
}

function renderMarkdown(workspace: WorkspaceRow, nodes: NodeRow[], includeEmptyNodes: boolean): string {
  const lines: string[] = [
    `# ${workspace.title}`,
    "",
    workspace.description ?? "",
    "",
  ]

  let sequence = 1
  for (const node of nodes) {
    const content = normalizeContent(node.content)
    if (!includeEmptyNodes && !content) {
      continue
    }

    lines.push(`## ${sequence}. ${node.title}`)
    lines.push(`> ${node.description ?? ""}`)
    if (content) {
      lines.push(content)
    }
    lines.push("")
    sequence += 1
  }

  return lines.join("\n").trim() + "\n"
}

function renderHtmlDocument(markdown: string, title: string): string {
  const escapedTitle = escapeHtml(title)
  const escapedMarkdown = escapeHtml(markdown)

  return [
    "<!DOCTYPE html>",
    '<html lang="zh-CN">',
    "<head>",
    '  <meta charset="UTF-8">',
    `  <title>${escapedTitle}</title>`,
    "  <style>",
    "    body { font-family: -apple-system, sans-serif; max-width: 800px; margin: 0 auto; padding: 2rem; color: #333; }",
    "    h1 { border-bottom: 2px solid #333; padding-bottom: 0.5rem; }",
    "    h2 { color: #555; margin-top: 2rem; }",
    "    blockquote { border-left: 3px solid #ccc; padding-left: 1rem; color: #666; }",
    "    @media print { body { padding: 0; } }",
    "  </style>",
    "</head>",
    "<body>",
    `  <pre style="white-space: pre-wrap;">${escapedMarkdown}</pre>`,
    "</body>",
    "</html>",
  ].join("\n")
}

async function resolveWorkspace(
  client: ReturnType<typeof createClient>,
  workshopId: string,
): Promise<WorkspaceRow | null> {
  const { data, error } = await client
    .schema("juanleme")
    .from("projects")
    .select("id, workspace_id, workspaces!inner(id, title, description)")
    .or(`id.eq.${workshopId},workspace_id.eq.${workshopId}`)
    .order("created_at", { ascending: true })
    .limit(1)

  if (error) {
    throw new Error(error.message)
  }

  const row = data?.[0] as
    | {
      id: string
      workspace_id: string
      workspaces: { id: string; title: string; description: string }
    }
    | undefined

  if (!row) {
    return null
  }

  return {
    project_id: row.id,
    workspace_id: row.workspace_id,
    title: row.workspaces.title,
    description: row.workspaces.description,
  }
}

async function ensureMembership(
  client: ReturnType<typeof createClient>,
  workspaceId: string,
  userId: string,
): Promise<boolean> {
  const { data, error } = await client
    .schema("juanleme")
    .from("workspace_memberships")
    .select("workspace_id")
    .eq("workspace_id", workspaceId)
    .eq("user_id", userId)
    .maybeSingle()

  if (error) {
    throw new Error(error.message)
  }

  return Boolean(data)
}

async function findIdempotentJob(
  client: ReturnType<typeof createClient>,
  workspaceId: string,
  userId: string,
  idempotencyKey: string,
): Promise<ExportJobRow | null> {
  const { data, error } = await client
    .schema("juanleme")
    .from("export_jobs")
    .select("id, status, format, created_at, artifact_url, artifact_path, error_message")
    .eq("workspace_id", workspaceId)
    .eq("user_id", userId)
    .eq("idempotency_key", idempotencyKey)
    .order("created_at", { ascending: false })
    .limit(1)
    .maybeSingle()

  if (error) {
    throw new Error(error.message)
  }

  return (data as ExportJobRow | null) ?? null
}

async function createExportJob(
  client: ReturnType<typeof createClient>,
  workspaceId: string,
  userId: string,
  format: ExportFormat,
  idempotencyKey?: string,
): Promise<ExportJobRow> {
  const payload = {
    workspace_id: workspaceId,
    user_id: userId,
    format,
    status: "pending",
    idempotency_key: idempotencyKey ?? null,
  }

  const attempt = async (dbFormat: string) =>
    client
      .schema("juanleme")
      .from("export_jobs")
      .insert({ ...payload, format: dbFormat })
      .select("id, status, format, created_at")
      .single()

  let insertResult = await attempt(format)

  if (
    insertResult.error &&
    format === "html" &&
    insertResult.error.message.toLowerCase().includes("check")
  ) {
    insertResult = await attempt("print")
  }

  if (insertResult.error || !insertResult.data) {
    throw new Error(insertResult.error?.message ?? "failed to create export job")
  }

  return insertResult.data as ExportJobRow
}

async function updateExportJobFailure(
  client: ReturnType<typeof createClient>,
  jobId: string,
  errorMessage: string,
): Promise<void> {
  const { error } = await client
    .schema("juanleme")
    .from("export_jobs")
    .update({
      status: "failed",
      error_message: errorMessage.slice(0, 2000),
      updated_at: new Date().toISOString(),
    })
    .eq("id", jobId)

  if (error) {
    throw new Error(error.message)
  }
}

async function updateExportJobSuccess(
  client: ReturnType<typeof createClient>,
  jobId: string,
  artifactPath: string,
): Promise<void> {
  const updateWithColumn = async (column: "artifact_url" | "artifact_path") =>
    client
      .schema("juanleme")
      .from("export_jobs")
      .update({
        status: "succeeded",
        [column]: artifactPath,
        error_message: null,
        updated_at: new Date().toISOString(),
      })
      .eq("id", jobId)

  let result = await updateWithColumn("artifact_url")
  if (result.error && result.error.message.toLowerCase().includes("artifact_url")) {
    result = await updateWithColumn("artifact_path")
  }

  if (result.error) {
    throw new Error(result.error.message)
  }
}

async function generateContent(
  client: ReturnType<typeof createClient>,
  workspaceId: string,
  userId: string,
  includeEmptyNodes: boolean,
  format: ExportFormat,
): Promise<{ content: string; title: string }> {
  const { data: workspaceData, error: workspaceError } = await client
    .schema("juanleme")
    .from("workspaces")
    .select("id, title, description")
    .eq("id", workspaceId)
    .single()

  if (workspaceError || !workspaceData) {
    throw new Error(workspaceError?.message ?? "workspace not found")
  }

  const workspace: WorkspaceRow = {
    project_id: "",
    workspace_id: workspaceData.id as string,
    title: (workspaceData.title as string) ?? "Workshop Export",
    description: (workspaceData.description as string) ?? "",
  }

  const { data: nodeRows, error: nodesError } = await client
    .schema("juanleme")
    .from("workshop_nodes")
    .select(`
      id,
      title,
      description,
      order,
      documents!left(content, user_id)
    `)
    .eq("workspace_id", workspaceId)
    .order("order", { ascending: true })

  if (nodesError) {
    throw new Error(nodesError.message)
  }

  const nodes = (nodeRows ?? []).map((row) => {
    const record = row as Record<string, unknown>
    const documents = (record.documents as Array<Record<string, unknown>> | null) ?? []
    const userDocument = documents.find((doc) => doc.user_id === userId)

    return {
      id: record.id as string,
      title: (record.title as string) ?? "Untitled",
      description: (record.description as string) ?? "",
      order: (record.order as number) ?? 0,
      content: userDocument?.content ?? null,
    } satisfies NodeRow
  })

  const markdown = renderMarkdown(workspace, nodes, includeEmptyNodes)
  if (format === "markdown") {
    return { content: markdown, title: workspace.title }
  }

  return {
    content: renderHtmlDocument(markdown, workspace.title),
    title: workspace.title,
  }
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

  let body: ExportRequest
  try {
    body = (await req.json()) as ExportRequest
  } catch {
    return jsonResponse({ error: "Invalid JSON body" }, 400)
  }

  const workshopId = body.workshopId?.trim()
  const format = body.format
  const includeEmptyNodes = body.includeEmptyNodes ?? false
  const idempotencyKey = body.idempotencyKey?.trim()

  if (!workshopId || (format !== "markdown" && format !== "html")) {
    return jsonResponse({ error: "workshopId and valid format are required" }, 400)
  }

  const {
    data: { user },
    error: userError,
  } = await jwtClient.auth.getUser()

  if (userError || !user) {
    return jsonResponse({ error: userError?.message ?? "Unauthorized" }, 401)
  }

  let workspace: WorkspaceRow | null = null
  try {
    workspace = await resolveWorkspace(jwtClient, workshopId)
  } catch (error) {
    const message = error instanceof Error ? error.message : "lineage validation failed"
    return jsonResponse({ error: message }, 500)
  }

  if (!workspace) {
    return jsonResponse({ error: "Workshop lineage not found" }, 404)
  }

  try {
    const isMember = await ensureMembership(jwtClient, workspace.workspace_id, user.id)
    if (!isMember) {
      return jsonResponse({ error: "Forbidden" }, 403)
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : "membership check failed"
    return jsonResponse({ error: message }, 500)
  }

  if (idempotencyKey) {
    try {
      const existingJob = await findIdempotentJob(
        serviceClient,
        workspace.workspace_id,
        user.id,
        idempotencyKey,
      )

      if (existingJob) {
        if (!isTerminalStatus(existingJob.status)) {
          return jsonResponse({ error: "Duplicate idempotency key is still pending" }, 409)
        }

        const artifactPath = existingJob.artifact_url ?? existingJob.artifact_path
        let downloadUrl: string | null = null

        if (artifactPath) {
          const { data: signed, error: signedError } = await serviceClient.storage
            .from("workshops")
            .createSignedUrl(artifactPath, 3600)

          if (!signedError) {
            downloadUrl = signed.signedUrl
          }
        }

        return jsonResponse({
          jobId: existingJob.id,
          status: existingJob.status,
          format,
          downloadUrl,
          createdAt: existingJob.created_at,
        })
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : "idempotency check failed"
      return jsonResponse({ error: message }, 500)
    }
  }

  let exportJob: ExportJobRow
  try {
    exportJob = await createExportJob(
      serviceClient,
      workspace.workspace_id,
      user.id,
      format,
      idempotencyKey,
    )
  } catch (error) {
    const message = error instanceof Error ? error.message : "failed to create export job"
    const code = message.includes("duplicate") ? 409 : 500
    return jsonResponse({ error: message }, code)
  }

  try {
    const { content } = await generateContent(
      jwtClient,
      workspace.workspace_id,
      user.id,
      includeEmptyNodes,
      format,
    )

    const extension = format === "markdown" ? "md" : "html"
    const fileName = `export.${extension}`
    const contentType = format === "markdown" ? "text/markdown; charset=utf-8" : "text/html; charset=utf-8"
    const artifactPath = `juanleme/exports/${workspace.workspace_id}/${exportJob.id}/${fileName}`

    const { error: uploadError } = await serviceClient.storage
      .from("workshops")
      .upload(artifactPath, new TextEncoder().encode(content), {
        contentType,
        upsert: true,
      })

    if (uploadError) {
      throw new Error(uploadError.message)
    }

    await updateExportJobSuccess(serviceClient, exportJob.id, artifactPath)

    const { data: signedData, error: signedError } = await serviceClient.storage
      .from("workshops")
      .createSignedUrl(artifactPath, 3600)

    if (signedError || !signedData?.signedUrl) {
      throw new Error(signedError?.message ?? "failed to create signed url")
    }

    return jsonResponse({
      jobId: exportJob.id,
      status: "succeeded",
      format,
      downloadUrl: signedData.signedUrl,
      createdAt: exportJob.created_at,
    })
  } catch (error) {
    const message = error instanceof Error ? error.message : "internal error"
    try {
      await updateExportJobFailure(serviceClient, exportJob.id, message)
    } catch {
      // 任务失败状态更新失败时不覆盖原始错误
    }
    return jsonResponse({ error: "Internal error", details: message }, 500)
  }
})
