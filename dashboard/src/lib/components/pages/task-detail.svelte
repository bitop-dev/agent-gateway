<script lang="ts">
  import { onMount } from "svelte";
  import { api, type Task } from "$lib/api";
  import * as Card from "$lib/components/ui/card/index.js";
  import { Badge } from "$lib/components/ui/badge/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Skeleton } from "$lib/components/ui/skeleton/index.js";
  import { navigate } from "$lib/router";
  import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
  import CopyIcon from "@lucide/svelte/icons/copy";

  let { taskId }: { taskId: string } = $props();

  let task = $state<Task | null>(null);
  let loading = $state(true);
  let error = $state("");

  async function loadTask(id: string) {
    loading = true;
    error = "";
    task = null;
    try {
      task = await api.getTask(id);
    } catch (e: any) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  onMount(() => loadTask(taskId));

  // Re-fetch when taskId changes (hash navigation)
  $effect(() => {
    loadTask(taskId);
  });

  function statusVariant(
    status: string
  ): "default" | "secondary" | "destructive" | "outline" {
    switch (status) {
      case "completed":
        return "default";
      case "failed":
        return "destructive";
      case "running":
        return "secondary";
      default:
        return "outline";
    }
  }

  function formatDuration(ms?: number): string {
    if (!ms) return "—";
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  }

  function formatTokens(n?: number): string {
    if (!n) return "—";
    return n.toLocaleString();
  }

  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text);
  }
</script>

<div class="space-y-6">
  <div class="flex items-center gap-4">
    <Button variant="ghost" size="sm" onclick={() => navigate("/tasks")}>
      <ArrowLeftIcon class="h-4 w-4 mr-1" />
      Back
    </Button>
    <h1 class="text-3xl font-bold">Task Detail</h1>
  </div>

  {#if loading}
    <Card.Root>
      <Card.Content class="space-y-4 p-6">
        {#each Array(6) as _}
          <Skeleton class="h-6 w-full" />
        {/each}
      </Card.Content>
    </Card.Root>
  {:else if error}
    <Card.Root>
      <Card.Content class="p-6">
        <p class="text-destructive">{error}</p>
      </Card.Content>
    </Card.Root>
  {:else if task}
    <!-- Metadata -->
    <Card.Root>
      <Card.Content class="p-6">
        <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <div>
            <p class="text-sm text-muted-foreground">Task ID</p>
            <p class="font-mono text-sm">{task.id}</p>
          </div>
          <div>
            <p class="text-sm text-muted-foreground">Profile</p>
            <p class="font-medium">{task.profile}</p>
          </div>
          <div>
            <p class="text-sm text-muted-foreground">Status</p>
            <Badge variant={statusVariant(task.status)}>{task.status}</Badge>
          </div>
          <div>
            <p class="text-sm text-muted-foreground">Duration</p>
            <p class="font-medium">{formatDuration(task.durationMs)}</p>
          </div>
          <div>
            <p class="text-sm text-muted-foreground">Worker</p>
            <p class="font-mono text-sm">{task.workerUrl || "—"}</p>
          </div>
          <div>
            <p class="text-sm text-muted-foreground">Model</p>
            <p class="font-medium">{task.model || "—"}</p>
          </div>
          <div>
            <p class="text-sm text-muted-foreground">Tokens</p>
            <p class="text-sm">
              {formatTokens(task.tokensIn)} in / {formatTokens(task.tokensOut)} out
            </p>
          </div>
          <div>
            <p class="text-sm text-muted-foreground">Cost</p>
            <p class="font-medium">${(task.cost || 0).toFixed(6)}</p>
          </div>
          <div>
            <p class="text-sm text-muted-foreground">Tool Calls</p>
            <p class="font-medium">{task.toolCalls ?? 0}</p>
          </div>
          <div>
            <p class="text-sm text-muted-foreground">Created</p>
            <p class="text-sm">
              {task.createdAt
                ? new Date(task.createdAt).toLocaleString()
                : "—"}
            </p>
          </div>
          <div>
            <p class="text-sm text-muted-foreground">Started</p>
            <p class="text-sm">
              {task.startedAt
                ? new Date(task.startedAt).toLocaleString()
                : "—"}
            </p>
          </div>
          <div>
            <p class="text-sm text-muted-foreground">Completed</p>
            <p class="text-sm">
              {task.completedAt
                ? new Date(task.completedAt).toLocaleString()
                : "—"}
            </p>
          </div>
        </div>
      </Card.Content>
    </Card.Root>

    <!-- Task prompt -->
    <Card.Root>
      <Card.Header>
        <Card.Title>Task</Card.Title>
      </Card.Header>
      <Card.Content>
        <div class="rounded-md bg-muted p-4">
          <p class="text-sm whitespace-pre-wrap">{task.task || "—"}</p>
        </div>
      </Card.Content>
    </Card.Root>

    <!-- Output -->
    {#if task.output}
      <Card.Root>
        <Card.Header>
          <div class="flex items-center justify-between">
            <Card.Title>Output</Card.Title>
            <Button
              variant="ghost"
              size="sm"
              onclick={() => copyToClipboard(task?.output || "")}
            >
              <CopyIcon class="h-4 w-4 mr-1" />
              Copy
            </Button>
          </div>
        </Card.Header>
        <Card.Content>
          <div class="rounded-md bg-muted p-4 text-sm whitespace-pre-wrap max-h-[600px] overflow-y-auto">
            {task.output}
          </div>
        </Card.Content>
      </Card.Root>
    {/if}

    <!-- Error -->
    {#if task.error}
      <Card.Root>
        <Card.Header>
          <Card.Title>Error</Card.Title>
        </Card.Header>
        <Card.Content>
          <div class="rounded-md bg-destructive/10 p-4 text-sm text-destructive whitespace-pre-wrap max-h-[600px] overflow-y-auto">
            {task.error}
          </div>
        </Card.Content>
      </Card.Root>
    {/if}
  {/if}
</div>
