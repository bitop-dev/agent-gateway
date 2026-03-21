<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { api, type Task } from "$lib/api";
  import { events } from "$lib/stores";
  import * as Card from "$lib/components/ui/card/index.js";
  import { Badge } from "$lib/components/ui/badge/index.js";
  import { Separator } from "$lib/components/ui/separator/index.js";
  import { Skeleton } from "$lib/components/ui/skeleton/index.js";
  import { navigate } from "$lib/router";
  import ServerIcon from "@lucide/svelte/icons/server";
  import ListTodoIcon from "@lucide/svelte/icons/list-todo";
  import CircleCheckIcon from "@lucide/svelte/icons/circle-check";
  import CircleXIcon from "@lucide/svelte/icons/circle-x";
  import DollarSignIcon from "@lucide/svelte/icons/dollar-sign";
  import ActivityIcon from "@lucide/svelte/icons/activity";

  let workers = $state(0);
  let totalTasks = $state(0);
  let completed = $state(0);
  let failed = $state(0);
  let totalCost = $state(0);
  let recentTasks = $state<Task[]>([]);
  let loading = $state(true);
  let interval: ReturnType<typeof setInterval>;

  async function refresh() {
    try {
      const [health, tasksResp, allTasks, costsResp] = await Promise.all([
        api.getHealth(),
        api.getTasks({ limit: 8 }),
        api.getTasks({ limit: 1000 }),
        api.getCosts(),
      ]);
      workers = health.workers;
      const all = allTasks.tasks || [];
      totalTasks = all.length;
      completed = all.filter((t) => t.status === "completed").length;
      failed = all.filter((t) => t.status === "failed").length;
      recentTasks = tasksResp.tasks || [];
      // costs: gateway returns { profiles: [...], totalCost }
      const costsData = costsResp as any;
      totalCost = costsData.totalCost || (costsData.costs || costsData.profiles || []).reduce(
        (sum: number, c: any) => sum + (c.cost || c.totalCost || 0),
        0
      );
    } catch (e) {
      console.error("Failed to fetch overview:", e);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    refresh();
    interval = setInterval(refresh, 10000);
  });

  onDestroy(() => clearInterval(interval));

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

  function formatDuration(ms?: number): string {
    if (!ms) return "";
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
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
</script>

<div class="space-y-6">
  <h1 class="text-3xl font-bold">Overview</h1>

  <!-- Stat cards -->
  <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
    {#if loading}
      {#each Array(5) as _}
        <Card.Root>
          <Card.Header class="flex flex-row items-center justify-between pb-2">
            <Skeleton class="h-4 w-20" />
            <Skeleton class="h-4 w-4" />
          </Card.Header>
          <Card.Content>
            <Skeleton class="h-8 w-16" />
          </Card.Content>
        </Card.Root>
      {/each}
    {:else}
      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between pb-2">
          <Card.Title class="text-sm font-medium">Workers</Card.Title>
          <ServerIcon class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{workers}</div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between pb-2">
          <Card.Title class="text-sm font-medium">Total Tasks</Card.Title>
          <ListTodoIcon class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{totalTasks}</div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between pb-2">
          <Card.Title class="text-sm font-medium">Completed</Card.Title>
          <CircleCheckIcon class="h-4 w-4 text-green-500" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold text-green-500">{completed}</div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between pb-2">
          <Card.Title class="text-sm font-medium">Failed</Card.Title>
          <CircleXIcon class="h-4 w-4 text-destructive" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold text-destructive">{failed}</div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between pb-2">
          <Card.Title class="text-sm font-medium">Total Cost</Card.Title>
          <DollarSignIcon class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">${totalCost.toFixed(4)}</div>
        </Card.Content>
      </Card.Root>
    {/if}
  </div>

  <div class="grid gap-6 lg:grid-cols-2">
    <!-- Recent tasks -->
    <Card.Root>
      <Card.Header>
        <div class="flex items-center justify-between">
          <Card.Title>Recent Tasks</Card.Title>
          <button
            class="text-sm text-muted-foreground hover:text-foreground"
            onclick={() => navigate("/tasks")}
          >
            View all →
          </button>
        </div>
      </Card.Header>
      <Card.Content>
        {#if loading}
          <div class="space-y-3">
            {#each Array(4) as _}
              <Skeleton class="h-10 w-full" />
            {/each}
          </div>
        {:else if recentTasks.length === 0}
          <p class="text-sm text-muted-foreground">No tasks yet</p>
        {:else}
          <div class="space-y-2">
            {#each recentTasks.slice(0, 6) as task (task.id)}
              <button
                class="flex w-full items-center justify-between rounded-md p-2 text-left hover:bg-accent"
                onclick={() => navigate(`/tasks/${task.id}`)}
              >
                <div class="flex items-center gap-3 min-w-0">
                  <Badge variant={statusVariant(task.status)}>
                    {task.status}
                  </Badge>
                  <span class="text-sm font-medium truncate"
                    >{task.profile}</span
                  >
                </div>
                <span class="text-xs text-muted-foreground whitespace-nowrap ml-2">
                  {timeAgo(task.createdAt)}
                </span>
              </button>
            {/each}
          </div>
        {/if}
      </Card.Content>
    </Card.Root>

    <!-- Live events -->
    <Card.Root>
      <Card.Header>
        <div class="flex items-center gap-2">
          <ActivityIcon class="h-4 w-4 text-green-500 animate-pulse" />
          <Card.Title>Live Events</Card.Title>
        </div>
      </Card.Header>
      <Card.Content>
        {#if $events.length === 0}
          <p class="text-sm text-muted-foreground">
            {#if !loading}
              Listening for events... Submit a task to see real-time activity.
            {:else}
              Connecting...
            {/if}
          </p>
        {:else}
          <div class="space-y-2 max-h-[300px] overflow-y-auto">
            {#each $events.slice(0, 10) as event (event.timestamp + event.type)}
              <div
                class="flex items-start gap-2 rounded-md p-2 text-sm bg-muted/50"
              >
                <span class="text-xs text-muted-foreground whitespace-nowrap font-mono">
                  {new Date(event.timestamp).toLocaleTimeString()}
                </span>
                <span class="font-medium">
                  {event.type.replace("agent.", "")}
                </span>
                {#if event.data?.profile}
                  <Badge variant="outline"
                    >{event.data.profile}</Badge
                  >
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      </Card.Content>
    </Card.Root>
  </div>
</div>
