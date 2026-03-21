<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { api, type Task } from "$lib/api";
  import * as Card from "$lib/components/ui/card/index.js";
  import * as Table from "$lib/components/ui/table/index.js";
  import { Badge } from "$lib/components/ui/badge/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Input } from "$lib/components/ui/input/index.js";
  import { Skeleton } from "$lib/components/ui/skeleton/index.js";
  import { navigate } from "$lib/router";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import PlusIcon from "@lucide/svelte/icons/plus";

  let tasks = $state<Task[]>([]);
  let loading = $state(true);
  let filter = $state("");
  let statusFilter = $state("");
  let interval: ReturnType<typeof setInterval>;

  async function refresh() {
    try {
      const opts: { limit: number; status?: string } = { limit: 100 };
      if (statusFilter) opts.status = statusFilter;
      const resp = await api.getTasks(opts);
      tasks = resp.tasks || [];
    } catch (e) {
      console.error("Failed to fetch tasks:", e);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    refresh();
    interval = setInterval(refresh, 10000);
  });

  onDestroy(() => clearInterval(interval));

  let filteredTasks = $derived(
    tasks.filter((t) => {
      if (!filter) return true;
      const q = filter.toLowerCase();
      return (
        t.id.toLowerCase().includes(q) ||
        t.profile.toLowerCase().includes(q) ||
        (t.task && t.task.toLowerCase().includes(q))
      );
    })
  );

  function timeAgo(ts: string): string {
    if (!ts) return "";
    const diff = Date.now() - new Date(ts).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return "just now";
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    return `${Math.floor(hours / 24)}d ago`;
  }

  function formatDuration(d?: number): string {
    if (!d) return "—";
    if (d < 1) return `${(d * 1000).toFixed(0)}ms`;
    return `${d.toFixed(1)}s`;
  }

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

  const statuses = ["", "queued", "running", "completed", "failed"];
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <h1 class="text-3xl font-bold">Tasks</h1>
    <div class="flex gap-2">
      <Button variant="outline" size="sm" onclick={refresh}>
        <RefreshCwIcon class="h-4 w-4" />
      </Button>
      <Button size="sm" onclick={() => navigate("/tasks/new")}>
        <PlusIcon class="h-4 w-4 mr-1" />
        New Task
      </Button>
    </div>
  </div>

  <!-- Filters -->
  <div class="flex gap-3">
    <Input
      placeholder="Search tasks..."
      class="max-w-sm"
      oninput={(e: Event) =>
        (filter = (e.target as HTMLInputElement).value)}
    />
    <div class="flex gap-1">
      {#each statuses as s}
        <Button
          variant={statusFilter === s ? "default" : "outline"}
          size="sm"
          onclick={() => {
            statusFilter = s;
            loading = true;
            refresh();
          }}
        >
          {s || "All"}
        </Button>
      {/each}
    </div>
  </div>

  <!-- Table -->
  <Card.Root>
    <Card.Content class="p-0">
      {#if loading}
        <div class="p-6 space-y-3">
          {#each Array(5) as _}
            <Skeleton class="h-10 w-full" />
          {/each}
        </div>
      {:else}
        <Table.Root>
          <Table.Header>
            <Table.Row>
              <Table.Head class="w-[140px]">ID</Table.Head>
              <Table.Head>Profile</Table.Head>
              <Table.Head>Task</Table.Head>
              <Table.Head class="w-[100px]">Status</Table.Head>
              <Table.Head class="w-[80px]">Duration</Table.Head>
              <Table.Head class="w-[100px] text-right">When</Table.Head>
            </Table.Row>
          </Table.Header>
          <Table.Body>
            {#each filteredTasks as task (task.id)}
              <Table.Row
                class="cursor-pointer hover:bg-accent/50"
                onclick={() => navigate(`/tasks/${task.id}`)}
              >
                <Table.Cell class="font-mono text-xs">
                  {task.id.slice(0, 14)}
                </Table.Cell>
                <Table.Cell class="font-medium">{task.profile}</Table.Cell>
                <Table.Cell class="max-w-[300px] truncate text-sm text-muted-foreground">
                  {task.task || "—"}
                </Table.Cell>
                <Table.Cell>
                  <Badge variant={statusVariant(task.status)}>
                    {task.status}
                  </Badge>
                </Table.Cell>
                <Table.Cell class="text-sm">
                  {formatDuration(task.duration)}
                </Table.Cell>
                <Table.Cell class="text-right text-xs text-muted-foreground">
                  {timeAgo(task.createdAt)}
                </Table.Cell>
              </Table.Row>
            {/each}
            {#if filteredTasks.length === 0}
              <Table.Row>
                <Table.Cell colspan={6} class="text-center py-8 text-muted-foreground">
                  No tasks found
                </Table.Cell>
              </Table.Row>
            {/if}
          </Table.Body>
        </Table.Root>
      {/if}
    </Card.Content>
  </Card.Root>
</div>
